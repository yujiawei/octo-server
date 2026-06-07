package usersecret

import (
	"testing"

	"github.com/Mininglamp-OSS/octo-lib/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// 跨表迁移依赖:resolve 反查 robot 表。
	_ "github.com/Mininglamp-OSS/octo-server/modules/robot"
)

func newTestSvc(t *testing.T) (*service, *store) {
	t.Helper()
	_, ctx := testutil.NewTestServer()
	require.NoError(t, testutil.CleanAllTables(ctx))
	st := newStore(ctx)
	enc, err := newEncryptor()
	require.NoError(t, err)
	return newService(st, enc), st
}

func TestService_CreateListResolve_Integration(t *testing.T) {
	svc, _ := newTestSvc(t)
	owner := "u-owner-1"

	view, err := svc.create(owner, "Claude 密钥", KindLLM, "sk-claude-xyz789")
	require.NoError(t, err)
	assert.NotEmpty(t, view.SecretID)
	assert.Equal(t, "Claude 密钥", view.DisplayName)
	assert.Equal(t, KindLLM, view.Kind)
	assert.Equal(t, "****z789", view.Masked)

	// list 脱敏:无明文/密文字段。
	views, err := svc.list(owner, "")
	require.NoError(t, err)
	require.Len(t, views, 1)
	assert.Equal(t, "****z789", views[0].Masked)

	// resolve by secret_id → 明文。
	out, err := svc.resolve(owner, view.SecretID)
	require.NoError(t, err)
	assert.Equal(t, resultOK, out.result)
	assert.Equal(t, "sk-claude-xyz789", out.plaintext)

	// resolve by display_name(精确)→ 明文。
	out, err = svc.resolve(owner, "claude密钥")
	require.NoError(t, err)
	assert.Equal(t, "sk-claude-xyz789", out.plaintext)
}

func TestService_DuplicateName_Integration(t *testing.T) {
	svc, _ := newTestSvc(t)
	owner := "u-owner-2"
	_, err := svc.create(owner, "OpenAI Key", KindLLM, "sk-a")
	require.NoError(t, err)
	// 归一化撞名(大小写 + 空格)。
	_, err = svc.create(owner, "openai  key", KindLLM, "sk-b")
	assert.ErrorIs(t, err, errDuplicateName)
}

func TestService_OwnerIsolation_Integration(t *testing.T) {
	svc, _ := newTestSvc(t)
	a, err := svc.create("u-a", "shared name", KindExternal, "secret-A")
	require.NoError(t, err)

	// 另一 owner 同名不冲突。
	_, err = svc.create("u-b", "shared name", KindExternal, "secret-B")
	require.NoError(t, err)

	// u-b 不能 resolve u-a 的 secret_id。
	out, err := svc.resolve("u-b", a.SecretID)
	assert.ErrorIs(t, err, errNotFound)
	assert.Equal(t, resultNotFound, out.result)
}

func TestService_UpdateKey_Integration(t *testing.T) {
	svc, _ := newTestSvc(t)
	owner := "u-upd"
	v, err := svc.create(owner, "rotate me", KindExternal, "old-value-1111")
	require.NoError(t, err)

	v2, err := svc.updateKey(owner, v.SecretID, "new-value-2222")
	require.NoError(t, err)
	assert.Equal(t, v.SecretID, v2.SecretID, "换 key 不改 secret_id")
	assert.Equal(t, "****2222", v2.Masked)

	out, err := svc.resolve(owner, v.SecretID)
	require.NoError(t, err)
	assert.Equal(t, "new-value-2222", out.plaintext)
}

func TestService_Rename_Integration(t *testing.T) {
	svc, _ := newTestSvc(t)
	owner := "u-ren"
	v, err := svc.create(owner, "old name", KindExternal, "val-keep")
	require.NoError(t, err)

	v2, err := svc.rename(owner, v.SecretID, "new name")
	require.NoError(t, err)
	assert.Equal(t, v.SecretID, v2.SecretID, "改名不断引用")
	assert.Equal(t, "new name", v2.DisplayName)

	// 密文不变:仍能解出原值。
	out, err := svc.resolve(owner, v.SecretID)
	require.NoError(t, err)
	assert.Equal(t, "val-keep", out.plaintext)

	// 改名后旧名解不到,新名解得到。
	_, err = svc.resolve(owner, "old name")
	assert.ErrorIs(t, err, errNotFound)
	out, err = svc.resolve(owner, "new name")
	require.NoError(t, err)
	assert.Equal(t, "val-keep", out.plaintext)
}

func TestService_Delete_Integration(t *testing.T) {
	svc, _ := newTestSvc(t)
	owner := "u-del"
	v, err := svc.create(owner, "to delete", KindExternal, "v")
	require.NoError(t, err)
	require.NoError(t, svc.delete(owner, v.SecretID))
	assert.ErrorIs(t, svc.delete(owner, v.SecretID), errNotFound)
}

func TestService_ResolveAmbiguous_Integration(t *testing.T) {
	svc, _ := newTestSvc(t)
	owner := "u-amb"
	// 两个拼音键相同(密钥/米要 → miyao)的别名,模糊查询应返候选而非明文。
	_, err := svc.create(owner, "我的密钥", KindExternal, "v1")
	require.NoError(t, err)
	_, err = svc.create(owner, "我的米要", KindExternal, "v2")
	require.NoError(t, err)

	out, err := svc.resolve(owner, "我的miyao")
	assert.ErrorIs(t, err, errAmbiguous)
	assert.Equal(t, resultAmbiguous, out.result)
	assert.Len(t, out.candidates, 2)
	assert.Empty(t, out.plaintext, "歧义不返明文")
}

func TestStore_QueryBotByToken_Integration(t *testing.T) {
	_, ctx := testutil.NewTestServer()
	require.NoError(t, testutil.CleanAllTables(ctx))
	st := newStore(ctx)

	_, err := ctx.DB().InsertBySql(
		"INSERT INTO robot (robot_id, creator_uid, bot_token, status) VALUES (?, ?, ?, 1)",
		"bot-1", "owner-77", "bf_token_abc",
	).Exec()
	require.NoError(t, err)

	id, err := st.queryBotByToken("bf_token_abc")
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.Equal(t, "bot-1", id.RobotID)
	assert.Equal(t, "owner-77", id.OwnerUID)

	// 未知 token → nil。
	id, err = st.queryBotByToken("bf_unknown")
	require.NoError(t, err)
	assert.Nil(t, id)
}
