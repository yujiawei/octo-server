package webhook

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Mininglamp-OSS/octo-lib/config"
	"github.com/Mininglamp-OSS/octo-server/modules/group"
)

// externalMarkerCacheDefaultTTL 是 GetMemberExternalMarkers + 群 owner_space 查询的
// 共享缓存过期时间。YUJ-172：push 热路径每条离线消息都会击中这两条 SQL，
// 加 5min TTL 内存缓存，命中率目标 > 90%。
const externalMarkerCacheDefaultTTL = 5 * time.Minute

// externalMarkerResolver 抽象 group 服务的依赖面，仅暴露 cache 需要的 2 个方法。
// 方便单元测试中注入 fake 实现，不依赖真实 DB。
type externalMarkerResolver interface {
	GetMemberExternalMarkers(groupNo string) (map[string]group.MemberExternalMarker, error)
	GetGroupWithGroupNo(groupNo string) (*group.InfoResp, error)
}

// externalMarkerEntry 是一个群的缓存快照：
//   - markers: uid -> MemberExternalMarker (含 home_space_id/name)
//   - groupSpaceID: 群本体的 owner_space_id，为空表示私有群/历史数据
//   - expiresAt: 该快照失效时间
type externalMarkerEntry struct {
	markers      map[string]group.MemberExternalMarker
	groupSpaceID string
	expiresAt    time.Time
}

// externalMarkerCache 是 groupNo -> entry 的带 TTL 的内存缓存。
// 所有公开方法并发安全。Hits / Misses 计数供观测 / 测试验证缓存命中率。
type externalMarkerCache struct {
	mu       sync.RWMutex
	entries  map[string]externalMarkerEntry
	ttl      time.Duration
	resolver externalMarkerResolver
	now      func() time.Time

	hits   int64
	misses int64
}

func newExternalMarkerCache(resolver externalMarkerResolver, ttl time.Duration) *externalMarkerCache {
	return &externalMarkerCache{
		entries:  make(map[string]externalMarkerEntry),
		ttl:      ttl,
		resolver: resolver,
		now:      time.Now,
	}
}

// Get 查询某个群某个成员的外部标识 + 群 owner_space_id。
// 返回值:
//   - marker: 该成员的外部标识；若该成员不存在/群未查到，返回零值并 exists=false
//   - groupSpaceID: 群本体的 owner_space_id
//   - exists: 成员是否在群内（缓存内有记录）
//   - err: 非 nil 表示底层 resolver 返回错误，调用方应降级处理
//
// groupNo 为空时直接返回零值，不触发缓存。
func (c *externalMarkerCache) Get(groupNo, fromUID string) (group.MemberExternalMarker, string, bool, error) {
	if strings.TrimSpace(groupNo) == "" {
		return group.MemberExternalMarker{}, "", false, nil
	}

	// 读快照
	c.mu.RLock()
	entry, ok := c.entries[groupNo]
	c.mu.RUnlock()

	now := c.now()
	if ok && entry.expiresAt.After(now) {
		atomic.AddInt64(&c.hits, 1)
		marker, exists := entry.markers[fromUID]
		return marker, entry.groupSpaceID, exists, nil
	}

	// miss: 回源
	markers, err := c.resolver.GetMemberExternalMarkers(groupNo)
	if err != nil {
		return group.MemberExternalMarker{}, "", false, err
	}
	groupInfo, err := c.resolver.GetGroupWithGroupNo(groupNo)
	if err != nil {
		return group.MemberExternalMarker{}, "", false, err
	}
	groupSpaceID := ""
	if groupInfo != nil {
		groupSpaceID = groupInfo.SpaceID
	}

	entry = externalMarkerEntry{
		markers:      markers,
		groupSpaceID: groupSpaceID,
		expiresAt:    c.now().Add(c.ttl),
	}
	c.mu.Lock()
	c.entries[groupNo] = entry
	c.mu.Unlock()
	atomic.AddInt64(&c.misses, 1)

	marker, exists := markers[fromUID]
	return marker, groupSpaceID, exists, nil
}

// Stats 返回 (hits, misses) 快照，供观测 / 单测断言。
func (c *externalMarkerCache) Stats() (int64, int64) {
	return atomic.LoadInt64(&c.hits), atomic.LoadInt64(&c.misses)
}

// Invalidate 主动清空指定群的缓存；为空串时清空所有。
// 目前未挂事件源，保留给未来的群成员变更 webhook 使用。
func (c *externalMarkerCache) Invalidate(groupNo string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if groupNo == "" {
		c.entries = make(map[string]externalMarkerEntry)
		return
	}
	delete(c.entries, groupNo)
}

// ---- 包级单例 ----

var (
	externalMarkerCacheInstance *externalMarkerCache
	externalMarkerCacheOnce     sync.Once
)

// getExternalMarkerCache 懒加载共享缓存，和 getWebhookDB 同款 sync.Once 模式。
func getExternalMarkerCache(ctx *config.Context) *externalMarkerCache {
	externalMarkerCacheOnce.Do(func() {
		externalMarkerCacheInstance = newExternalMarkerCache(
			group.NewService(ctx),
			externalMarkerCacheDefaultTTL,
		)
	})
	return externalMarkerCacheInstance
}

// resolveSenderSpaceLabel 根据群号 + 发件人 UID 判断是否需要在推送 body
// 里追加 `@SpaceName` 外部标识。返回空串表示不需要追加（同 Space / 数据缺失 / 降级）。
//
// 规则（对齐 YUJ-172 / octo-server#1251）：
//  1. 群号或 UID 为空 → 不追加
//  2. 发件人不在成员快照里（脏数据）→ 不追加
//  3. 群本体 owner_space_id 或发件人 home_space_id 为空 → 不追加（保守降级）
//  4. 发件人 home_space_id == 群 owner_space_id → 同 Space，不追加
//  5. 否则追加 home_space_name（空则降级为不追加）
func resolveSenderSpaceLabel(cache *externalMarkerCache, groupNo, fromUID string) string {
	if cache == nil || groupNo == "" || fromUID == "" {
		return ""
	}
	marker, groupSpaceID, exists, err := cache.Get(groupNo, fromUID)
	if err != nil || !exists {
		return ""
	}
	if groupSpaceID == "" || marker.HomeSpaceID == "" {
		return ""
	}
	if marker.HomeSpaceID == groupSpaceID {
		return ""
	}
	return sanitizeSpaceLabel(marker.HomeSpaceName)
}

// sanitizeSpaceLabel 对 space_name 做最小保守编码：
// 1) 去 HTML 尖括号（YUJ-172 硬约束：push body 不进 DOM，但保守防注入）
// 2) 去换行/回车（避免把推送 body 截断成多行）
// 3) 去首尾空白
func sanitizeSpaceLabel(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.NewReplacer(
		"<", "",
		">", "",
		"\r", "",
		"\n", " ",
	).Replace(s)
	return s
}
