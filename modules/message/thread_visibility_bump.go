package message

// Per-user thread visibility follow_version bump helpers (plan T5/T6).
//
// 背景：客户端只观察 follow_version 感知「自己的 follow 列表/可见性变了」。per-user
// 归档相关写路径（写 reminder_done / 写 reminder / 写 archive_intent）都要在事务里
// bump 被影响 uid 的 follow_version，客户端才会重新拉 sidebar 跑仲裁。
//
// 锁序铁律（plan F3/R3）：新写路径同 tx 若碰 conversation_ext，必须先对 user_follow_version
// 加锁（即先调 BumpFollowVersionTx）再碰 user_conversation_ext。这里只 bump，不碰 ext 行，
// 但仍统一走 BumpFollowVersionTx 以复用其 INSERT ... ON DUPLICATE KEY 幂等语义。

import (
	"github.com/Mininglamp-OSS/octo-lib/common"
	convext "github.com/Mininglamp-OSS/octo-server/modules/conversation_ext"
	"github.com/Mininglamp-OSS/octo-server/modules/thread"
	"github.com/gocraft/dbr/v2"
	"go.uber.org/zap"
)

// resolveSpaceIDForGroup 把 groupNo 解析成 space_id（复用 groupService.GetGroupWithGroupNo）。
// 群不存在 / 无 space_id 时返回空串（follow_version 表按 (uid, space_id) 建行，空 space 亦合法）。
func (m *Message) resolveSpaceIDForGroup(groupNo string) string {
	if groupNo == "" {
		return ""
	}
	info, err := m.groupService.GetGroupWithGroupNo(groupNo)
	if err != nil || info == nil {
		return ""
	}
	return info.SpaceID
}

// resolveBumpSpaceForUID 解析「某 uid 在某群下」follow_version 应落的 space（P1 修复）。
//
// 背景（YUJ-8101 P1-1）：sidebar 读侧按 **request space** 读 follow_version
// （api_sidebar.go GetSpaceID(c) → followVersionDB.Get(loginUID, spaceID)）——
// 对跨 space 外部成员（group_member.is_external=1, source_space_id）来说，request space
// 就是他自己的 source space，不是群的 home space。若 bump 一律落群 home space，
// 外部成员的 bump 落在 (uid, group-space) 而读在 (uid, source-space)，follow_version
// 永不推进，per-user 归档翻转 + P1 强制拉回信号对他不生效。
//
// 修法镜像既有成例 conversation_ext.OnThreadCreated（在各自 ext-row space bump）：
//   - 外部成员（is_external=1）→ 其 source_space_id（与读侧同分区）；
//   - 内部成员（含成员行查不到 / 查询失败）→ 群 home space（与旧行为一致，保守回落）。
func (m *Message) resolveBumpSpaceForUID(groupNo, uid string) string {
	groupSpace := m.resolveSpaceIDForGroup(groupNo)
	if groupNo == "" || uid == "" {
		return groupSpace
	}
	isExternal, sourceSpaceID, _, _, _, err := m.groupService.GetMemberExternalFields(groupNo, uid)
	if err != nil {
		// 查询失败：保守回落群 home space（等价旧行为），不阻断 bump。
		return groupSpace
	}
	if isExternal == 1 {
		return sourceSpaceID // 外部成员读侧按 source space —— bump 必须落同一分区
	}
	return groupSpace
}

// bumpFollowVersionForThreadChannelsTx 对一批 thread channel_id（"{groupNo}____{shortID}"）
// 解析出 space_id 后，在同一 tx 内 bump (uid, space_id) 的 follow_version（去重每个 space 只 bump 一次）。
// 用于 reminder_done 侧（T5）。channel_id 解析失败的条目跳过（不阻断核心写）。
func (m *Message) bumpFollowVersionForThreadChannelsTx(tx *dbr.Tx, uid string, channelIDs []string) error {
	if uid == "" || len(channelIDs) == 0 {
		return nil
	}
	bumped := make(map[string]struct{})
	for _, ch := range channelIDs {
		groupNo, _, err := thread.ParseChannelID(ch)
		if err != nil {
			continue
		}
		// P1-1：按「该 uid 在该群」的读侧分区解析 bump space（外部成员→source space），
		// 而非群 home space，否则跨 space 成员的 bump 与读落在不同 (uid, space) 分区。
		spaceID := m.resolveBumpSpaceForUID(groupNo, uid)
		if _, ok := bumped[spaceID]; ok {
			continue
		}
		if _, err := convext.BumpFollowVersionTx(tx, uid, spaceID); err != nil {
			return err
		}
		bumped[spaceID] = struct{}{}
	}
	return nil
}

// bumpFollowVersionForReminders 是 reminder 触发侧 bump（plan T6）的同步内核：
// 对每条 per-uid（uid≠”）且属于子区（channel_type=5）的新 reminder，bump 被@uid 的
// follow_version。@所有人(uid=”)不 bump。
//
// 关键约束（plan T6/R2，顾问共识）：**绝不把本 bump 包进 handleReminders 的核心写 tx**。
// 消息入库是高频核心链路，bump 是边缘 UI 链路。这里每个 (uid, space) 各自独立短 tx、
// 失败只 warn，绝不影响消息投递。调用方（handleReminders）以 fire-and-forget goroutine 调用它。
func (m *Message) bumpFollowVersionForReminders(reminders []*remindersModel) {
	if !thread.PerUserVisibilityEnabled() || len(reminders) == 0 {
		return
	}
	// fire-and-forget 在独立 goroutine 里跑：任何 panic 都必须就地 recover，否则会拖垮整个进程
	// （与 remindersDB.inserts / reminderDone 的 recover 惯例一致）。
	defer func() {
		if r := recover(); r != nil {
			m.Warn("T6 bump: recovered panic (non-fatal)", zap.Any("panic", r))
		}
	}()
	// 去重 (uid, space)：同一被@人在同 space 多子区只 bump 一次。
	type key struct{ uid, space string }
	seen := make(map[key]struct{})
	for _, r := range reminders {
		if r == nil || r.UID == "" { // 广播 uid='' 不 bump
			continue
		}
		if r.ChannelType != uint8(common.ChannelTypeCommunityTopic) {
			continue // 只处理子区 reminder
		}
		groupNo, _, err := thread.ParseChannelID(r.ChannelID)
		if err != nil {
			continue
		}
		// P1-1：按被@ uid 的读侧分区解析 bump space（外部成员→source space）。
		spaceID := m.resolveBumpSpaceForUID(groupNo, r.UID)
		k := key{uid: r.UID, space: spaceID}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}

		// 各自独立短 tx，失败只 warn（fire-and-forget 语义，不阻断消息投递）。
		// 包一层闭包让 defer tx.RollbackUnlessCommitted() 每次迭代就地生效（P2-b）：
		// 顶层 recover 只 log 不 rollback，Begin 与 Commit 之间若 driver panic 会泄漏 tx；
		// per-iteration defer 兜底回滚，与 reminderDone 的 recover+rollback 成例对齐。
		func(uid, spaceID string) {
			tx, err := m.ctx.DB().Begin()
			if err != nil {
				m.Warn("T6 bump: begin tx failed (non-fatal)", zap.Error(err), zap.String("uid", uid))
				return
			}
			defer tx.RollbackUnlessCommitted()
			if _, err := convext.BumpFollowVersionTx(tx, uid, spaceID); err != nil {
				m.Warn("T6 bump: BumpFollowVersionTx failed (non-fatal)", zap.Error(err), zap.String("uid", uid))
				return
			}
			if err := tx.Commit(); err != nil {
				m.Warn("T6 bump: commit failed (non-fatal)", zap.Error(err), zap.String("uid", uid))
			}
		}(r.UID, spaceID)
	}
}
