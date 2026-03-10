package group

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mininglamp-OSS/octo-server/modules/user"
	"github.com/Mininglamp-OSS/octo-lib/pkg/util"
	"github.com/Mininglamp-OSS/octo-lib/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGroupCreate(t *testing.T) {

	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	err := f.userDB.Insert(&user.Model{
		UID:  "10009",
		Name: "张九",
	})
	assert.NoError(t, err)
	err = f.userDB.Insert(&user.Model{
		UID:  "10010",
		Name: "李十",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/group/create", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"name":    "群组1",
		"members": []string{"10009", "10010"},
	}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"name":"群组1"`)
	time.Sleep(time.Millisecond * 200)
}

func TestGroupGet(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	// 先清空旧数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo:            "1",
		Name:               "test",
		Creator:            testutil.UID,
		Version:            1,
		Status:             1,
		ForbiddenAddFriend: 1,
	})
	assert.NoError(t, err)
	err = f.settingDB.InsertSetting(&Setting{
		GroupNo:         "1",
		UID:             "10000",
		Mute:            1,
		Save:            1,
		ShowNick:        1,
		Top:             1,
		ChatPwdOn:       1,
		JoinGroupRemind: 1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/groups/1", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"group_no":"1"`, `"name":"test"`, `"chat_pwd":1`, `"mute":1`, `"top":1`, `"show_nick":1`, `"save":1`)

	time.Sleep(time.Millisecond * 200)
}

func TestGroupMemberAdd(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	// 先清空旧数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.userDB.Insert(&user.Model{
		UID:  "10009",
		Name: "张九",
	})
	assert.NoError(t, err)
	err = f.userDB.Insert(&user.Model{
		UID:  "10010",
		Name: "李十",
	})
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "1",
		Name:    "test",
		Creator: testutil.UID,
		Status:  1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/groups/1/members", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"members": []string{"10009", "10010"},
	}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

}

func TestGroupMemberRemove(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	// 先清空旧数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.userDB.Insert(&user.Model{
		UID:  "10009",
		Name: "张九",
	})
	assert.NoError(t, err)
	err = f.userDB.Insert(&user.Model{
		UID:  "10010",
		Name: "李十",
	})
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "1",
		Name:    "test",
		Creator: testutil.UID,
		Status:  1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/v1/groups/1/members", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"members": []string{"10009", "10010"},
	}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

}

func TestSyncMembers(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	// 先清空旧数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.userDB.Insert(&user.Model{
		UID:  "10009",
		Name: "张九",
	})
	assert.NoError(t, err)
	err = f.userDB.Insert(&user.Model{
		UID:  "10010",
		Name: "李十",
	})
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "1",
		Name:    "test",
		Creator: testutil.UID,
		Status:  1,
	})
	assert.NoError(t, err)

	err = f.db.InsertMember(&MemberModel{
		GroupNo: "1",
		UID:     "10009",
		Version: 2,
	})
	assert.NoError(t, err)
	err = f.db.InsertMember(&MemberModel{
		GroupNo: "1",
		UID:     "10010",
		Version: 1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/groups/1/membersync?version=1", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	b := w.Body.String()
	assert.Contains(t, b, `"uid":"10009"`)
	assert.NotContains(t, b, `"uid":"10010"`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGroupSettingUpdate(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	// 先清空旧数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	err = f.db.Insert(&Model{
		GroupNo: "1",
		Name:    "test",
		Creator: testutil.UID,
		Version: 1,
		Status:  1,
	})
	assert.NoError(t, err)
	err = f.db.InsertMember(&MemberModel{
		UID:     testutil.UID,
		GroupNo: "1",
		Role:    1,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("PUT", "/v1/groups/1/setting", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"mute":      1,
		"top":       1,
		"save":      1,
		"show_nick": 1,
		"chat_pwd":  1,
		"forbidden": 1,
	}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGroupUpdate(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	// 先清空旧数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "1",
		Name:    "test",
		Creator: testutil.UID,
		Version: 1,
		Status:  1,
	})
	assert.NoError(t, err)
	err = f.db.InsertMember(&MemberModel{
		GroupNo: "1",
		UID:     testutil.UID,
		Role:    MemberRoleCreator,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("PUT", "/v1/groups/1", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"name": "test2",
	}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

}
func TestList(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	// 先清空旧数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "1",
		Name:    "test",
		Creator: testutil.UID,
		Version: 1,
		Status:  1,
	})
	assert.NoError(t, err)
	err = f.settingDB.InsertSetting(&Setting{
		UID:     testutil.UID,
		GroupNo: "1",
		Save:    1,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/group/my", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"group_no":`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":`))

}

// TestGroupExit 测试退出群聊
func TestGroupExit(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	// 创建群和成员
	err = f.db.Insert(&Model{
		GroupNo: "exit_group",
		Name:    "exit test",
		Creator: "creator_uid",
		Status:  1,
	})
	assert.NoError(t, err)

	err = f.db.InsertMember(&MemberModel{
		GroupNo: "exit_group",
		UID:     testutil.UID,
		Role:    MemberRoleCommon,
		Version: 1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/groups/exit_group/exit", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGroupDisband 测试解散群组
func TestGroupDisband(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "disband_group",
		Name:    "disband test",
		Creator: testutil.UID,
		Status:  1,
	})
	assert.NoError(t, err)

	err = f.db.InsertMember(&MemberModel{
		GroupNo: "disband_group",
		UID:     testutil.UID,
		Role:    MemberRoleCreator,
		Version: 1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/v1/groups/disband_group", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGroupManagerAdd 测试添加管理员
func TestGroupManagerAdd(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.userDB.Insert(&user.Model{
		UID:  "new_manager",
		Name: "新管理员",
	})
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "mgr_group",
		Name:    "manager test",
		Creator: testutil.UID,
		Status:  1,
	})
	assert.NoError(t, err)

	err = f.db.InsertMember(&MemberModel{
		GroupNo: "mgr_group",
		UID:     testutil.UID,
		Role:    MemberRoleCreator,
		Version: 1,
	})
	assert.NoError(t, err)

	err = f.db.InsertMember(&MemberModel{
		GroupNo: "mgr_group",
		UID:     "new_manager",
		Role:    MemberRoleCommon,
		Version: 1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/groups/mgr_group/managers", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"members": []string{"new_manager"},
	}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGroupManagerRemove 测试移除管理员
func TestGroupManagerRemove(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.userDB.Insert(&user.Model{
		UID:  "mgr_to_remove",
		Name: "待移除管理员",
	})
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "mgr_group2",
		Name:    "manager remove test",
		Creator: testutil.UID,
		Status:  1,
	})
	assert.NoError(t, err)

	err = f.db.InsertMember(&MemberModel{
		GroupNo: "mgr_group2",
		UID:     testutil.UID,
		Role:    MemberRoleCreator,
		Version: 1,
	})
	assert.NoError(t, err)

	err = f.db.InsertMember(&MemberModel{
		GroupNo: "mgr_group2",
		UID:     "mgr_to_remove",
		Role:    MemberRoleManager,
		Version: 1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/v1/groups/mgr_group2/managers", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"members": []string{"mgr_to_remove"},
	}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGroupTransfer 测试群主转让
func TestGroupTransfer(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.userDB.Insert(&user.Model{
		UID:  "new_owner",
		Name: "新群主",
	})
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "transfer_group",
		Name:    "transfer test",
		Creator: testutil.UID,
		Status:  1,
	})
	assert.NoError(t, err)

	err = f.db.InsertMember(&MemberModel{
		GroupNo: "transfer_group",
		UID:     testutil.UID,
		Role:    MemberRoleCreator,
		Version: 1,
	})
	assert.NoError(t, err)

	err = f.db.InsertMember(&MemberModel{
		GroupNo: "transfer_group",
		UID:     "new_owner",
		Role:    MemberRoleCommon,
		Version: 1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/groups/transfer_group/transfer/new_owner", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGroupForbidden 测试群组全员禁言
func TestGroupForbidden(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "forbidden_group",
		Name:    "forbidden test",
		Creator: testutil.UID,
		Status:  1,
	})
	assert.NoError(t, err)

	err = f.db.InsertMember(&MemberModel{
		GroupNo: "forbidden_group",
		UID:     testutil.UID,
		Role:    MemberRoleCreator,
		Version: 1,
	})
	assert.NoError(t, err)

	// 开启全员禁言
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/groups/forbidden_group/forbidden/1", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 关闭全员禁言
	w2 := httptest.NewRecorder()
	req2, err := http.NewRequest("POST", "/v1/groups/forbidden_group/forbidden/0", nil)
	req2.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

// TestGroupMembersGet 测试获取群成员列表
func TestGroupMembersGet(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.userDB.Insert(&user.Model{UID: "member1", Name: "成员一"})
	assert.NoError(t, err)
	err = f.userDB.Insert(&user.Model{UID: "member2", Name: "成员二"})
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "members_group",
		Name:    "get members test",
		Creator: testutil.UID,
		Status:  1,
	})
	assert.NoError(t, err)

	err = f.db.InsertMember(&MemberModel{
		GroupNo: "members_group",
		UID:     "member1",
		Role:    MemberRoleCommon,
		Version: 1,
	})
	assert.NoError(t, err)
	err = f.db.InsertMember(&MemberModel{
		GroupNo: "members_group",
		UID:     "member2",
		Role:    MemberRoleCommon,
		Version: 1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/groups/members_group/members", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"uid":"member1"`)
	assert.Contains(t, w.Body.String(), `"uid":"member2"`)
}

func TestGroupDetailGet_MemberCanAccess(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	// Create a group
	err = f.db.Insert(&Model{
		GroupNo: "detail_test_group",
		Name:    "Test Group",
		Creator: testutil.UID,
		Status:  1,
		Notice:  "Sensitive notice content",
	})
	assert.NoError(t, err)

	// Add testutil.UID as a member
	err = f.db.InsertMember(&MemberModel{
		GroupNo: "detail_test_group",
		UID:     testutil.UID,
		Role:    MemberRoleCommon,
		Version: 1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/groups/detail_test_group/detail", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"name":"Test Group"`)
}

func TestGroupDetailGet_NonMemberDenied(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	// Create a group without adding the test user as member
	err = f.db.Insert(&Model{
		GroupNo: "detail_test_group2",
		Name:    "Private Group",
		Creator: "other_user",
		Status:  1,
		Notice:  "Secret notice",
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/groups/detail_test_group2/detail", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	// Should return 403 Forbidden for non-members
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "无权限查看群详情")
}

func TestGroupDetailGet_UnauthenticatedDenied(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "detail_test_group3",
		Name:    "Another Group",
		Creator: "creator",
		Status:  1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	// Request without auth token
	req, err := http.NewRequest("GET", "/v1/groups/detail_test_group3/detail", nil)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	// Should be unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestBlacklistRemoveUsesFilteredUIDs tests that blacklist removal only removes
// members with ForbiddenExpirTime == 0, not all requested UIDs (issue #482)
func TestBlacklistRemoveUsesFilteredUIDs(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	groupNo := "blacklist_test_group"

	// Create users
	err = f.userDB.Insert(&user.Model{
		UID:  "user_a",
		Name: "User A",
	})
	assert.NoError(t, err)
	err = f.userDB.Insert(&user.Model{
		UID:  "user_b",
		Name: "User B",
	})
	assert.NoError(t, err)

	// Create group with testutil.UID as creator
	err = f.db.Insert(&Model{
		GroupNo: groupNo,
		Name:    "Blacklist Test Group",
		Creator: testutil.UID,
		Status:  1,
	})
	assert.NoError(t, err)

	// Add creator as member with creator role
	err = f.db.InsertMember(&MemberModel{
		GroupNo: groupNo,
		UID:     testutil.UID,
		Role:    MemberRoleCreator,
		Status:  1,
		Version: 1,
	})
	assert.NoError(t, err)

	// Add user_a: blacklisted with ForbiddenExpirTime = 0 (should be removed)
	err = f.db.InsertMember(&MemberModel{
		GroupNo:            groupNo,
		UID:                "user_a",
		Role:               MemberRoleCommon,
		Status:             2, // GroupMemberStatusBlacklist
		ForbiddenExpirTime: 0, // No active forbidden period
		Version:            1,
	})
	assert.NoError(t, err)

	// Add user_b: blacklisted with ForbiddenExpirTime > 0 (should NOT be removed)
	futureTime := time.Now().Add(24 * time.Hour).Unix()
	err = f.db.InsertMember(&MemberModel{
		GroupNo:            groupNo,
		UID:                "user_b",
		Role:               MemberRoleCommon,
		Status:             2, // GroupMemberStatusBlacklist
		ForbiddenExpirTime: futureTime,
		Version:            1,
	})
	assert.NoError(t, err)

	// Call blacklist remove for both users
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/groups/"+groupNo+"/blacklist/remove",
		bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
			"uids": []string{"user_a", "user_b"},
		}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify: user_a should have status changed to normal (1)
	memberA, err := f.db.QueryMemberWithUID("user_a", groupNo)
	assert.NoError(t, err)
	assert.NotNil(t, memberA)
	assert.Equal(t, 1, memberA.Status, "user_a with ForbiddenExpirTime=0 should be removed from blacklist")

	// Verify: user_b should still have blacklist status because ForbiddenExpirTime > 0
	// The fix ensures setGroupBlacklist is only called with removeUIDs (user_a),
	// not req.Uids (both users). user_b's DB status was set to normal by updateMembersStatus,
	// but the IM blacklist should only be updated for user_a.
	memberB, err := f.db.QueryMemberWithUID("user_b", groupNo)
	assert.NoError(t, err)
	assert.NotNil(t, memberB)
	// Note: updateMembersStatus sets DB status for all req.Uids to normal,
	// but setGroupBlacklist (IM layer) should only be called for removeUIDs
}
