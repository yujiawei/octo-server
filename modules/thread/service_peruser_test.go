package thread

import (
	"strings"
	"testing"

	"github.com/Mininglamp-OSS/octo-lib/common"
	"github.com/Mininglamp-OSS/octo-lib/config"
	"github.com/Mininglamp-OSS/octo-lib/pkg/util"
	"github.com/Mininglamp-OSS/octo-lib/testutil"
	convext "github.com/Mininglamp-OSS/octo-server/modules/conversation_ext"
	"github.com/Mininglamp-OSS/octo-server/modules/group"
	"github.com/Mininglamp-OSS/octo-server/modules/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedThreadReminder inserts an unhandled per-uid @ (P1) reminder for a thread channel.
func seedThreadReminder(t *testing.T, ctx *config.Context, uid, groupNo, shortID string) {
	t.Helper()
	channelID := groupNo + "____" + shortID
	_, err := ctx.DB().InsertBySql(
		"INSERT INTO reminders (channel_id, channel_type, reminder_type, uid, is_deleted, version) VALUES (?,?,?,?,0,1)",
		channelID, uint8(common.ChannelTypeCommunityTopic), 1, uid,
	).Exec()
	require.NoError(t, err)
}

// seedActiveThread inserts an active thread row directly (avoids CreateThread's live-IM dep,
// which is why the DB-heavy service tests are otherwise skipped). Uses a snowflake-shaped shortID.
func seedActiveThread(t *testing.T, svc *Service, groupNo, shortID string) {
	t.Helper()
	require.NoError(t, svc.db.Insert(&Model{
		ShortID:    shortID,
		GroupNo:    groupNo,
		Name:       "thr-" + shortID,
		CreatorUID: testutil.UID,
		Status:     ThreadStatusActive,
		Version:    1,
	}))
}

const testShortID = "170000000000001" // 15-digit snowflake-shaped shortID (creator can operate)

// TestArchiveThread_PerUser_FlagOn 覆盖 plan T7 + T8 + gate 1/5：
// flag=on 时 ArchiveThread 写 per-uid intent（不改全局 status），A 视角归档、他人不变，且 bump follow_version；
// detail 同源。
func TestArchiveThread_PerUser_FlagOn(t *testing.T) {
	t.Setenv("DM_THREAD_PERUSER_VISIBILITY", "true")
	svc, groupNo := setupServiceTestData(t)
	shortID := testShortID
	seedActiveThread(t, svc, groupNo, shortID)

	fvDB := convext.NewFollowVersionDB(svc.ctx)
	before, err := fvDB.Get(testutil.UID, "")
	require.NoError(t, err)

	// A（creator）归档：写 per-uid intent，全局 status 不变。
	require.NoError(t, svc.ArchiveThread(groupNo, shortID, testutil.UID))

	globalRow, err := svc.db.QueryByGroupNoAndShortID(groupNo, shortID)
	require.NoError(t, err)
	assert.Equal(t, ThreadStatusActive, globalRow.Status, "flag=on 不改全局 status")

	states, err := svc.db.QueryUserStates(testutil.UID, []ShortRef{{GroupNo: groupNo, ShortID: shortID}})
	require.NoError(t, err)
	require.Contains(t, states, groupNo+"____"+shortID)
	assert.Equal(t, 1, states[groupNo+"____"+shortID].ArchiveIntent)

	otherStates, err := svc.db.QueryUserStates("user2", []ShortRef{{GroupNo: groupNo, ShortID: shortID}})
	require.NoError(t, err)
	assert.Empty(t, otherStates, "他人不受影响")

	after, err := fvDB.Get(testutil.UID, "")
	require.NoError(t, err)
	assert.Equal(t, before+1, after, "archive per-uid bumps follow_version")

	// detail 同源（T8）：A 视角 detail.status == 2；user2 仍 active。
	resp, err := svc.GetThread(groupNo, shortID, testutil.UID)
	require.NoError(t, err)
	assert.Equal(t, ThreadStatusArchived, resp.Status, "detail per-uid archived for A")

	respOther, err := svc.GetThread(groupNo, shortID, "user2")
	require.NoError(t, err)
	assert.Equal(t, ThreadStatusActive, respOther.Status, "detail active for others")

	// Unarchive 恢复：intent=0。
	require.NoError(t, svc.UnarchiveThread(groupNo, shortID, testutil.UID))
	states, err = svc.db.QueryUserStates(testutil.UID, []ShortRef{{GroupNo: groupNo, ShortID: shortID}})
	require.NoError(t, err)
	assert.Equal(t, 0, states[groupNo+"____"+shortID].ArchiveIntent, "unarchive sets intent 0")
	resp, err = svc.GetThread(groupNo, shortID, testutil.UID)
	require.NoError(t, err)
	assert.Equal(t, ThreadStatusActive, resp.Status)
}

// TestArchiveThread_Global_FlagOff 覆盖 gate 5：flag=off 时走原全局 CAS（现状），不写 per-uid 表。
func TestArchiveThread_Global_FlagOff(t *testing.T) {
	t.Setenv("DM_THREAD_PERUSER_VISIBILITY", "false")
	svc, groupNo := setupServiceTestData(t)
	shortID := testShortID
	seedActiveThread(t, svc, groupNo, shortID)

	require.NoError(t, svc.ArchiveThread(groupNo, shortID, testutil.UID))

	globalRow, err := svc.db.QueryByGroupNoAndShortID(groupNo, shortID)
	require.NoError(t, err)
	assert.Equal(t, ThreadStatusArchived, globalRow.Status, "flag=off 改全局 status（现状）")

	states, err := svc.db.QueryUserStates(testutil.UID, []ShortRef{{GroupNo: groupNo, ShortID: shortID}})
	require.NoError(t, err)
	assert.Empty(t, states, "flag=off 不写 thread_user_state")

	resp, err := svc.GetThread(groupNo, shortID, testutil.UID)
	require.NoError(t, err)
	assert.Equal(t, ThreadStatusArchived, resp.Status)
}

// TestGetThread_P1OverIntent 覆盖 T8 + gate 1：detail 端 P1 压过 intent 重新可见。
func TestGetThread_P1OverIntent(t *testing.T) {
	t.Setenv("DM_THREAD_PERUSER_VISIBILITY", "true")
	svc, groupNo := setupServiceTestData(t)
	shortID := testShortID
	seedActiveThread(t, svc, groupNo, shortID)

	require.NoError(t, svc.ArchiveThread(groupNo, shortID, testutil.UID))
	resp, err := svc.GetThread(groupNo, shortID, testutil.UID)
	require.NoError(t, err)
	require.Equal(t, ThreadStatusArchived, resp.Status)

	// 给 A 一个未处理 per-uid @（P1）→ detail 拉回 active。
	seedThreadReminder(t, svc.ctx, testutil.UID, groupNo, shortID)
	resp, err = svc.GetThread(groupNo, shortID, testutil.UID)
	require.NoError(t, err)
	assert.Equal(t, ThreadStatusActive, resp.Status, "P1 pulls archived thread back to active in detail")
}

// TestDeleteThread_GC 覆盖 T-GC：DeleteThread 后 user_state 行被清（无孤儿）。
func TestDeleteThread_GC(t *testing.T) {
	t.Setenv("DM_THREAD_PERUSER_VISIBILITY", "true")
	svc, groupNo := setupServiceTestData(t)
	shortID := testShortID
	seedActiveThread(t, svc, groupNo, shortID)

	require.NoError(t, svc.ArchiveThread(groupNo, shortID, testutil.UID))
	states, err := svc.db.QueryUserStates(testutil.UID, []ShortRef{{GroupNo: groupNo, ShortID: shortID}})
	require.NoError(t, err)
	require.NotEmpty(t, states)

	require.NoError(t, svc.DeleteThread(groupNo, shortID, testutil.UID))
	states, err = svc.db.QueryUserStates(testutil.UID, []ShortRef{{GroupNo: groupNo, ShortID: shortID}})
	require.NoError(t, err)
	assert.Empty(t, states, "DeleteThread cleans thread_user_state (no orphan)")
}

// =============================================================================
// P1-2: fail-open 真失败分支覆盖（YUJ-8148）
//
// effectiveStatusForUser（detail 端仲裁）有三个 error 分支：mute 查询 / user_state
// 查询 / P1 查询。原测试只覆盖 loginUID=="" 早返回与 0 行成功查询，从未强制这三个
// query 真报错。这里用 RENAME TABLE ..._bak 让每个 backing 表在查询时缺失（原子、
// defer 还原、不污染其它测试；与 user/incomingwebhook fail-secure 测试同法），断言
// query 报错时 detail 保留**全局 status**（fail-open，绝不误藏）。
//
// 这些测试无 build tag，在 CI gate 内跑。
// =============================================================================

// renameTableAway 把某表重命名走开，返回还原函数。查询该表的语句会因表缺失而报错，
// 从而驱动 fail-open error 分支。CleanAllTables 只 TRUNCATE 不 DROP，故不会误删 _bak。
func renameTableAway(t *testing.T, svc *Service, table string) func() {
	t.Helper()
	bak := table + "_failopen_bak"
	_, err := svc.ctx.DB().Exec("RENAME TABLE " + table + " TO " + bak)
	require.NoError(t, err)
	return func() {
		_, _ = svc.ctx.DB().Exec("RENAME TABLE " + bak + " TO " + table)
	}
}

// TestEffectiveStatus_FailOpen_MuteQueryError 覆盖 mute 查询 error 分支：
// thread_setting 表缺失 → QuerySetting 真报错 → 保留全局 status（active），不隐藏。
func TestEffectiveStatus_FailOpen_MuteQueryError(t *testing.T) {
	t.Setenv("DM_THREAD_PERUSER_VISIBILITY", "true")
	svc, groupNo := setupServiceTestData(t)
	shortID := testShortID
	seedActiveThread(t, svc, groupNo, shortID)

	restore := renameTableAway(t, svc, "thread_setting")
	defer restore()

	got := svc.effectiveStatusForUser(testutil.UID, groupNo, shortID, ThreadStatusActive)
	assert.Equal(t, ThreadStatusActive, got,
		"mute 查询报错时保留全局 status（fail-open，不隐藏）")

	// 全局 archived 时同样必须原样保留（不因查询失败误判成 active/隐藏）。
	got = svc.effectiveStatusForUser(testutil.UID, groupNo, shortID, ThreadStatusArchived)
	assert.Equal(t, ThreadStatusArchived, got,
		"mute 查询报错时保留全局 archived status")
}

// TestEffectiveStatus_FailOpen_UserStateQueryError 覆盖 user_state 查询 error 分支：
// thread_user_state 表缺失 → QueryUserStates 真报错 → 保留全局 status，不隐藏。
func TestEffectiveStatus_FailOpen_UserStateQueryError(t *testing.T) {
	t.Setenv("DM_THREAD_PERUSER_VISIBILITY", "true")
	svc, groupNo := setupServiceTestData(t)
	shortID := testShortID
	seedActiveThread(t, svc, groupNo, shortID)

	restore := renameTableAway(t, svc, "thread_user_state")
	defer restore()

	got := svc.effectiveStatusForUser(testutil.UID, groupNo, shortID, ThreadStatusArchived)
	assert.Equal(t, ThreadStatusArchived, got,
		"user_state 查询报错时保留全局 status（fail-open，不隐藏）")
}

// TestEffectiveStatus_FailOpen_P1QueryError 覆盖 P1 查询 error 分支：
// reminders 表缺失 → HasUnhandledMention 真报错 → 保留全局 status，不隐藏。
func TestEffectiveStatus_FailOpen_P1QueryError(t *testing.T) {
	t.Setenv("DM_THREAD_PERUSER_VISIBILITY", "true")
	svc, groupNo := setupServiceTestData(t)
	shortID := testShortID
	seedActiveThread(t, svc, groupNo, shortID)

	restore := renameTableAway(t, svc, "reminders")
	defer restore()

	got := svc.effectiveStatusForUser(testutil.UID, groupNo, shortID, ThreadStatusArchived)
	assert.Equal(t, ThreadStatusArchived, got,
		"P1 查询报错时保留全局 status（fail-open，不隐藏）")
}

// TestEffectiveStatus_FailOpen_ViaGetThread 端到端验证 fail-open：flag=on 下
// GetThread 路由到 effectiveStatusForUser，user_state 表缺失时 detail 仍返回全局
// status 而非 500 / 隐藏，与 sidebar 读侧 fail-open 同口径。
func TestEffectiveStatus_FailOpen_ViaGetThread(t *testing.T) {
	t.Setenv("DM_THREAD_PERUSER_VISIBILITY", "true")
	svc, groupNo := setupServiceTestData(t)
	shortID := testShortID
	seedActiveThread(t, svc, groupNo, shortID)

	restore := renameTableAway(t, svc, "thread_user_state")
	defer restore()

	resp, err := svc.GetThread(groupNo, shortID, testutil.UID)
	require.NoError(t, err, "detail 查询失败必须 fail-open，不冒泡成错误")
	assert.Equal(t, ThreadStatusActive, resp.Status,
		"user_state 查询报错时 detail 保留全局 status（active）")
}

// =============================================================================
// P1-1: 跨 space follow_version bump 落点（YUJ-8148）
//
// 修复前：setArchiveIntentPerUser 一律按群 home space bump；外部成员读侧按其
// source space 读 → bump 落 (uid, group-space)、读在 (uid, source-space)，follow_version
// 永不推进。修复后按 operatorUID 解析（external→source space），bump 落读侧同一分区。
// =============================================================================

// TestArchiveBump_ExternalMember_LandsInSourceSpace 覆盖 gate 4：外部成员归档的
// follow_version bump 落在其 source space（读侧同分区），不落群 home space。
func TestArchiveBump_ExternalMember_LandsInSourceSpace(t *testing.T) {
	t.Setenv("DM_THREAD_PERUSER_VISIBILITY", "true")
	_, ctx := testutil.NewTestServer()
	require.NoError(t, testutil.CleanAllTables(ctx))

	const (
		extUID      = "ext_user"
		groupSpace  = "space_home"   // 群 home space
		sourceSpace = "space_source" // 外部成员来源 space（读侧分区）
	)
	shortID := testShortID

	// 用户
	userDB := user.NewDB(ctx)
	require.NoError(t, userDB.Insert(&user.Model{UID: extUID, Name: "外部用户", ShortNo: "uext01"}))

	// 群（home space = groupSpace）
	groupNo := strings.ReplaceAll(util.GenerUUID(), "-", "")
	groupDB := group.NewDB(ctx)
	require.NoError(t, groupDB.Insert(&group.Model{
		GroupNo: groupNo, Name: "跨space测试群", Creator: extUID, Status: 1, Version: 1, SpaceID: groupSpace,
	}))
	// 外部成员：is_external=1, source_space_id=sourceSpace，角色 creator（可归档）。
	require.NoError(t, groupDB.InsertMember(&group.MemberModel{
		GroupNo: groupNo, UID: extUID, Role: group.MemberRoleCreator, Status: 1, Version: 1,
		Vercode: util.GenerUUID(), IsExternal: 1, SourceSpaceID: sourceSpace,
	}))

	svc := NewService(ctx).(*Service)
	// 子区创建者 = extUID，canOperate 才放行归档。
	require.NoError(t, svc.db.Insert(&Model{
		ShortID: shortID, GroupNo: groupNo, Name: "thr-" + shortID,
		CreatorUID: extUID, Status: ThreadStatusActive, Version: 1,
	}))

	fvDB := convext.NewFollowVersionDB(svc.ctx)
	homeBefore, err := fvDB.Get(extUID, groupSpace)
	require.NoError(t, err)
	sourceBefore, err := fvDB.Get(extUID, sourceSpace)
	require.NoError(t, err)

	// 外部成员归档子区。
	require.NoError(t, svc.ArchiveThread(groupNo, shortID, extUID))

	homeAfter, err := fvDB.Get(extUID, groupSpace)
	require.NoError(t, err)
	sourceAfter, err := fvDB.Get(extUID, sourceSpace)
	require.NoError(t, err)

	// 自证（gate 4 落点对比）：bump 落 source space（读侧同分区），群 home space 不推进。
	assert.Equal(t, sourceBefore+1, sourceAfter,
		"外部成员归档 bump 必须落在其 source space（读侧同分区）")
	assert.Equal(t, homeBefore, homeAfter,
		"外部成员归档 bump 不应落在群 home space（修复前的错误落点）")
}
