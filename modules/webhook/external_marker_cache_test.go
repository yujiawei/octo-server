package webhook

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Mininglamp-OSS/octo-server/modules/group"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeMarkerResolver 是 externalMarkerResolver 的测试替身，
// 记录调用次数，支持按群号返回预设的 markers / groupInfo / error。
type fakeMarkerResolver struct {
	mu             sync.Mutex
	markersByGroup map[string]map[string]group.MemberExternalMarker
	infoByGroup    map[string]*group.InfoResp
	markersErr     map[string]error
	infoErr        map[string]error

	markersCalls int
	infoCalls    int
}

func newFakeMarkerResolver() *fakeMarkerResolver {
	return &fakeMarkerResolver{
		markersByGroup: make(map[string]map[string]group.MemberExternalMarker),
		infoByGroup:    make(map[string]*group.InfoResp),
		markersErr:     make(map[string]error),
		infoErr:        make(map[string]error),
	}
}

func (f *fakeMarkerResolver) GetMemberExternalMarkers(groupNo string) (map[string]group.MemberExternalMarker, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.markersCalls++
	if err, ok := f.markersErr[groupNo]; ok {
		return nil, err
	}
	return f.markersByGroup[groupNo], nil
}

func (f *fakeMarkerResolver) GetGroupWithGroupNo(groupNo string) (*group.InfoResp, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.infoCalls++
	if err, ok := f.infoErr[groupNo]; ok {
		return nil, err
	}
	return f.infoByGroup[groupNo], nil
}

// TestExternalMarkerCache_MissThenHit 验证首次 Get 触发回源，
// 第二次 Get 在 TTL 内直接命中缓存，resolver 只被调用一次。
func TestExternalMarkerCache_MissThenHit(t *testing.T) {
	resolver := newFakeMarkerResolver()
	resolver.markersByGroup["g1"] = map[string]group.MemberExternalMarker{
		"userA": {IsExternal: 1, SourceSpaceName: "空间A", HomeSpaceID: "spA", HomeSpaceName: "空间A"},
	}
	resolver.infoByGroup["g1"] = &group.InfoResp{GroupNo: "g1", SpaceID: "spB"}

	cache := newExternalMarkerCache(resolver, 5*time.Minute)

	// 第一次：miss → 回源
	marker, groupSpaceID, exists, err := cache.Get("g1", "userA")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, 1, marker.IsExternal)
	assert.Equal(t, "空间A", marker.HomeSpaceName)
	assert.Equal(t, "spB", groupSpaceID)
	hits, misses := cache.Stats()
	assert.Equal(t, int64(0), hits)
	assert.Equal(t, int64(1), misses)
	assert.Equal(t, 1, resolver.markersCalls)
	assert.Equal(t, 1, resolver.infoCalls)

	// 第二次：hit → 不回源
	marker2, groupSpaceID2, exists2, err := cache.Get("g1", "userA")
	require.NoError(t, err)
	assert.True(t, exists2)
	assert.Equal(t, marker, marker2)
	assert.Equal(t, groupSpaceID, groupSpaceID2)
	hits, misses = cache.Stats()
	assert.Equal(t, int64(1), hits)
	assert.Equal(t, int64(1), misses)
	assert.Equal(t, 1, resolver.markersCalls, "缓存命中时不应再调 resolver")
	assert.Equal(t, 1, resolver.infoCalls, "缓存命中时不应再调 resolver")
}

// TestExternalMarkerCache_TTLExpiry 验证条目过期后再次 Get 触发回源。
func TestExternalMarkerCache_TTLExpiry(t *testing.T) {
	resolver := newFakeMarkerResolver()
	resolver.markersByGroup["g1"] = map[string]group.MemberExternalMarker{
		"userA": {IsExternal: 0, HomeSpaceID: "spB", HomeSpaceName: "空间B"},
	}
	resolver.infoByGroup["g1"] = &group.InfoResp{GroupNo: "g1", SpaceID: "spB"}

	cache := newExternalMarkerCache(resolver, 100*time.Millisecond)
	fakeNow := time.Now()
	cache.now = func() time.Time { return fakeNow }

	_, _, _, err := cache.Get("g1", "userA")
	require.NoError(t, err)
	assert.Equal(t, 1, resolver.markersCalls)

	// 推进时间超过 TTL → 下一次 Get 应重新回源
	fakeNow = fakeNow.Add(200 * time.Millisecond)
	_, _, _, err = cache.Get("g1", "userA")
	require.NoError(t, err)
	assert.Equal(t, 2, resolver.markersCalls)
	assert.Equal(t, 2, resolver.infoCalls)
}

// TestExternalMarkerCache_EmptyGroupNo 验证空群号直接返回零值，不触发 resolver。
func TestExternalMarkerCache_EmptyGroupNo(t *testing.T) {
	resolver := newFakeMarkerResolver()
	cache := newExternalMarkerCache(resolver, time.Minute)

	marker, groupSpaceID, exists, err := cache.Get("", "userA")
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Empty(t, groupSpaceID)
	assert.Equal(t, group.MemberExternalMarker{}, marker)
	assert.Equal(t, 0, resolver.markersCalls)
	assert.Equal(t, 0, resolver.infoCalls)
}

// TestExternalMarkerCache_MarkersError 验证 GetMemberExternalMarkers 报错时不缓存，
// 调用方能拿到错误并降级。
func TestExternalMarkerCache_MarkersError(t *testing.T) {
	resolver := newFakeMarkerResolver()
	resolver.markersErr["g1"] = errors.New("db down")

	cache := newExternalMarkerCache(resolver, time.Minute)

	_, _, _, err := cache.Get("g1", "userA")
	assert.Error(t, err)
	// 下次调用仍会尝试回源（没有缓存毒药）
	_, _, _, err = cache.Get("g1", "userA")
	assert.Error(t, err)
	assert.Equal(t, 2, resolver.markersCalls)
}

// TestExternalMarkerCache_Invalidate 验证 Invalidate 使指定群号下次 Get 重新回源。
func TestExternalMarkerCache_Invalidate(t *testing.T) {
	resolver := newFakeMarkerResolver()
	resolver.markersByGroup["g1"] = map[string]group.MemberExternalMarker{
		"userA": {IsExternal: 0, HomeSpaceID: "spB", HomeSpaceName: "空间B"},
	}
	resolver.infoByGroup["g1"] = &group.InfoResp{GroupNo: "g1", SpaceID: "spB"}

	cache := newExternalMarkerCache(resolver, time.Minute)
	_, _, _, err := cache.Get("g1", "userA")
	require.NoError(t, err)
	_, _, _, err = cache.Get("g1", "userA")
	require.NoError(t, err)
	assert.Equal(t, 1, resolver.markersCalls)

	cache.Invalidate("g1")
	_, _, _, err = cache.Get("g1", "userA")
	require.NoError(t, err)
	assert.Equal(t, 2, resolver.markersCalls)

	cache.Invalidate("")
	_, _, _, err = cache.Get("g1", "userA")
	require.NoError(t, err)
	assert.Equal(t, 3, resolver.markersCalls)
}

// TestResolveSenderSpaceLabel_CrossSpace 发件人 home_space != 群 owner_space →
// 返回 home_space_name 作为 @SpaceName 后缀。
func TestResolveSenderSpaceLabel_CrossSpace(t *testing.T) {
	resolver := newFakeMarkerResolver()
	resolver.markersByGroup["g1"] = map[string]group.MemberExternalMarker{
		"lisi": {IsExternal: 1, SourceSpaceName: "空间A", HomeSpaceID: "spA", HomeSpaceName: "空间A"},
	}
	resolver.infoByGroup["g1"] = &group.InfoResp{GroupNo: "g1", SpaceID: "spB"}

	cache := newExternalMarkerCache(resolver, time.Minute)
	label := resolveSenderSpaceLabel(cache, "g1", "lisi")
	assert.Equal(t, "空间A", label)
}

// TestResolveSenderSpaceLabel_SameSpace 同 Space 发件人 → 无后缀。
func TestResolveSenderSpaceLabel_SameSpace(t *testing.T) {
	resolver := newFakeMarkerResolver()
	resolver.markersByGroup["g1"] = map[string]group.MemberExternalMarker{
		"lisi": {IsExternal: 0, HomeSpaceID: "spB", HomeSpaceName: "空间B"},
	}
	resolver.infoByGroup["g1"] = &group.InfoResp{GroupNo: "g1", SpaceID: "spB"}

	cache := newExternalMarkerCache(resolver, time.Minute)
	assert.Equal(t, "", resolveSenderSpaceLabel(cache, "g1", "lisi"))
}

// TestResolveSenderSpaceLabel_MissingMember 发件人不在群成员快照里（脏数据）→ 无后缀。
func TestResolveSenderSpaceLabel_MissingMember(t *testing.T) {
	resolver := newFakeMarkerResolver()
	resolver.markersByGroup["g1"] = map[string]group.MemberExternalMarker{}
	resolver.infoByGroup["g1"] = &group.InfoResp{GroupNo: "g1", SpaceID: "spB"}

	cache := newExternalMarkerCache(resolver, time.Minute)
	assert.Equal(t, "", resolveSenderSpaceLabel(cache, "g1", "ghost"))
}

// TestResolveSenderSpaceLabel_EmptyGroupSpace 群 owner_space_id 为空（私有群 / 历史数据）→ 无后缀。
func TestResolveSenderSpaceLabel_EmptyGroupSpace(t *testing.T) {
	resolver := newFakeMarkerResolver()
	resolver.markersByGroup["g1"] = map[string]group.MemberExternalMarker{
		"lisi": {IsExternal: 1, HomeSpaceID: "spA", HomeSpaceName: "空间A"},
	}
	resolver.infoByGroup["g1"] = &group.InfoResp{GroupNo: "g1", SpaceID: ""}

	cache := newExternalMarkerCache(resolver, time.Minute)
	assert.Equal(t, "", resolveSenderSpaceLabel(cache, "g1", "lisi"))
}

// TestResolveSenderSpaceLabel_EmptyHomeSpace 发件人 home_space_id 为空 → 无后缀（保守降级）。
func TestResolveSenderSpaceLabel_EmptyHomeSpace(t *testing.T) {
	resolver := newFakeMarkerResolver()
	resolver.markersByGroup["g1"] = map[string]group.MemberExternalMarker{
		"lisi": {IsExternal: 1, HomeSpaceID: "", HomeSpaceName: ""},
	}
	resolver.infoByGroup["g1"] = &group.InfoResp{GroupNo: "g1", SpaceID: "spB"}

	cache := newExternalMarkerCache(resolver, time.Minute)
	assert.Equal(t, "", resolveSenderSpaceLabel(cache, "g1", "lisi"))
}

// TestResolveSenderSpaceLabel_ResolverError resolver 报错 → 无后缀（推送不阻断）。
func TestResolveSenderSpaceLabel_ResolverError(t *testing.T) {
	resolver := newFakeMarkerResolver()
	resolver.markersErr["g1"] = errors.New("db down")

	cache := newExternalMarkerCache(resolver, time.Minute)
	assert.Equal(t, "", resolveSenderSpaceLabel(cache, "g1", "lisi"))
}

// TestResolveSenderSpaceLabel_NilCache 防御性：nil cache → 无后缀，不 panic。
func TestResolveSenderSpaceLabel_NilCache(t *testing.T) {
	assert.Equal(t, "", resolveSenderSpaceLabel(nil, "g1", "lisi"))
}

// TestSanitizeSpaceLabel 验证 space_name 的保守编码。
func TestSanitizeSpaceLabel(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"普通中文", "销售部", "销售部"},
		{"去首尾空白", "  空间A  ", "空间A"},
		{"去 HTML 尖括号", "<script>evil</script>", "scriptevil/script"},
		{"去换行", "line1\nline2", "line1 line2"},
		{"去回车", "cr\rhere", "crhere"},
		{"全空白降级", "   ", ""},
		{"空串降级", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeSpaceLabel(tt.in))
		})
	}
}

// TestExternalMarkerCache_ConcurrentSafe 基本并发压测：
// 100 goroutine 并发 Get 同一个群，不应 panic / race。
func TestExternalMarkerCache_ConcurrentSafe(t *testing.T) {
	resolver := newFakeMarkerResolver()
	resolver.markersByGroup["g1"] = map[string]group.MemberExternalMarker{
		"userA": {IsExternal: 1, HomeSpaceID: "spA", HomeSpaceName: "空间A"},
	}
	resolver.infoByGroup["g1"] = &group.InfoResp{GroupNo: "g1", SpaceID: "spB"}

	cache := newExternalMarkerCache(resolver, time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _, _ = cache.Get("g1", "userA")
		}()
	}
	wg.Wait()
	hits, misses := cache.Stats()
	assert.Equal(t, int64(100), hits+misses, "每个 goroutine 都应记录一次 hit 或 miss")
	assert.GreaterOrEqual(t, hits, int64(1), "至少应该有一次 hit 路径被触发")
}
