package message

import (
	"testing"

	spacepkg "github.com/Mininglamp-OSS/octo-server/pkg/space"
	"github.com/Mininglamp-OSS/octo-lib/common"
	"github.com/stretchr/testify/assert"
)

func TestFilterConversationsBySpace_DirectMatch(t *testing.T) {
	// 会话 SpaceID 直接匹配 filterSpaceID → 保留
	convs := []*SyncUserConversationResp{
		{ChannelID: "g1", ChannelType: common.ChannelTypeGroup.Uint8(), SpaceID: "spaceA"},
		{ChannelID: "g2", ChannelType: common.ChannelTypeGroup.Uint8(), SpaceID: "spaceB"},
		{ChannelID: "u1", ChannelType: common.ChannelTypePerson.Uint8(), SpaceID: "spaceA"},
	}

	// 所有会话都有 SpaceID，不触发 bareGroupNos / bareDMUIDs 逻辑
	result := filterConversationsCore(convs, "spaceA", "spaceA", nil, nil, nil, false, false)
	assert.Len(t, result, 2)
	assert.Equal(t, "g1", result[0].ChannelID)
	assert.Equal(t, "u1", result[1].ChannelID)
}

func TestFilterConversationsBySpace_SystemBotsVisible(t *testing.T) {
	// 系统 Bot 应在所有 Space 可见（非默认 Space 中的裸 DM）
	convs := []*SyncUserConversationResp{
		{ChannelID: "botfather", ChannelType: common.ChannelTypePerson.Uint8(), SpaceID: ""},
		{ChannelID: "u_10000", ChannelType: common.ChannelTypePerson.Uint8(), SpaceID: ""},
		{ChannelID: "fileHelper", ChannelType: common.ChannelTypePerson.Uint8(), SpaceID: ""},
		{ChannelID: "custom_bot", ChannelType: common.ChannelTypePerson.Uint8(), SpaceID: ""},
	}

	// filterSpaceID != defaultSpaceID，所以走"非默认 Space 中的 DM"分支
	result := filterConversationsCore(convs, "spaceB", "spaceA", nil, nil, nil, false, false)
	// 系统 Bot 可见，custom_bot 不可见
	assert.Len(t, result, 3)
	ids := []string{result[0].ChannelID, result[1].ChannelID, result[2].ChannelID}
	assert.Contains(t, ids, "botfather")
	assert.Contains(t, ids, "u_10000")
	assert.Contains(t, ids, "fileHelper")
}

func TestFilterConversationsBySpace_DefaultSpaceBareConvs(t *testing.T) {
	// 裸 UID 旧会话只在默认 Space 显示
	convs := []*SyncUserConversationResp{
		{ChannelID: "user1", ChannelType: common.ChannelTypePerson.Uint8(), SpaceID: ""},
		{ChannelID: "user2", ChannelType: common.ChannelTypePerson.Uint8(), SpaceID: ""},
	}

	// filterSpaceID == defaultSpaceID
	result := filterConversationsCore(convs, "spaceA", "spaceA", nil, nil, nil, false, false)
	assert.Len(t, result, 2)

	// filterSpaceID != defaultSpaceID → 不显示普通 DM
	result = filterConversationsCore(convs, "spaceB", "spaceA", nil, nil, nil, false, false)
	assert.Len(t, result, 0)
}

func TestFilterConversationsBySpace_GroupSpaceMap(t *testing.T) {
	// 群聊通过 groupSpaceMap 匹配
	convs := []*SyncUserConversationResp{
		{ChannelID: "g1", ChannelType: common.ChannelTypeGroup.Uint8(), SpaceID: ""},
		{ChannelID: "g2", ChannelType: common.ChannelTypeGroup.Uint8(), SpaceID: ""},
	}
	groupMap := map[string]string{"g1": "spaceA", "g2": "spaceB"}

	result := filterConversationsCore(convs, "spaceA", "spaceA", groupMap, nil, nil, false, false)
	assert.Len(t, result, 1)
	assert.Equal(t, "g1", result[0].ChannelID)
}

func TestFilterConversationsBySpace_SkipGroupFilter(t *testing.T) {
	// skipGroupFilter=true 时保留所有裸群聊
	convs := []*SyncUserConversationResp{
		{ChannelID: "g1", ChannelType: common.ChannelTypeGroup.Uint8(), SpaceID: ""},
		{ChannelID: "g2", ChannelType: common.ChannelTypeGroup.Uint8(), SpaceID: ""},
		{ChannelID: "u1", ChannelType: common.ChannelTypePerson.Uint8(), SpaceID: "spaceA"},
	}

	result := filterConversationsCore(convs, "spaceA", "spaceA", nil, nil, nil, true, false)
	// g1, g2 保留（skipGroupFilter），u1 保留（直接匹配）
	assert.Len(t, result, 3)
}

func TestGetBotUIDs_SkipsSystemBots(t *testing.T) {
	// 系统 Bot 不应被 GetBotUIDs 查询（它们在调用前被排除）
	uids := []string{"botfather", "u_10000", "fileHelper"}
	for _, uid := range uids {
		assert.True(t, spacepkg.SystemBots[uid], "should be system bot: %s", uid)
	}
}

func TestGetGroupSpaceMap_Empty(t *testing.T) {
	result, err := spacepkg.GetGroupSpaceMap(nil, func(nos []string) ([]spacepkg.GroupSpaceInfo, error) {
		t.Fatal("should not be called for empty input")
		return nil, nil
	})
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetGroupSpaceMap_Maps(t *testing.T) {
	result, err := spacepkg.GetGroupSpaceMap([]string{"g1", "g2"}, func(nos []string) ([]spacepkg.GroupSpaceInfo, error) {
		return []spacepkg.GroupSpaceInfo{
			{GroupNo: "g1", SpaceID: "spaceA"},
			{GroupNo: "g2", SpaceID: "spaceB"},
		}, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "spaceA", result["g1"])
	assert.Equal(t, "spaceB", result["g2"])
}

func TestCheckBotsInSpace_EmptyInputs(t *testing.T) {
	// Empty spaceID → empty result without DB call
	result, err := spacepkg.CheckBotsInSpace(nil, "", map[string]bool{"bot1": true})
	assert.NoError(t, err)
	assert.Empty(t, result)

	// Empty botUIDs → empty result without DB call
	result, err = spacepkg.CheckBotsInSpace(nil, "spaceA", map[string]bool{})
	assert.NoError(t, err)
	assert.Empty(t, result)
}
