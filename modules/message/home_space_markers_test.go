package message

import (
	"testing"

	"github.com/Mininglamp-OSS/octo-server/modules/group"
	"github.com/stretchr/testify/assert"
)

// TestSyncMessages_ExternalMember_HomeSpace 外部成员发送的消息：
// from_home_space_id = 来源 space，from_home_space_name = 来源 space name（YUJ-63 / #1208）。
func TestSyncMessages_ExternalMember_HomeSpace(t *testing.T) {
	markers := map[string]group.MemberExternalMarker{
		"ext-uid": {
			IsExternal:      1,
			SourceSpaceName: "ExampleCorp",
			HomeSpaceID:     "space_example",
			HomeSpaceName:   "ExampleCorp",
		},
	}
	messages := []*MsgSyncResp{{FromUID: "ext-uid"}}

	applyExternalMarkers(messages, markers)

	// 原语义保留
	assert.Equal(t, 1, messages[0].FromIsExternal)
	assert.Equal(t, "ExampleCorp", messages[0].FromSourceSpaceName)
	// 新字段：外部成员的 home 等于来源 space
	assert.Equal(t, "space_example", messages[0].FromHomeSpaceID,
		"外部成员 from_home_space_id 应等于 source_space_id")
	assert.Equal(t, "ExampleCorp", messages[0].FromHomeSpaceName,
		"外部成员 from_home_space_name 应等于 source_space_name")
}

// TestSyncMessages_InternalMember_HomeSpace 内部成员发送的消息：
// from_home_space_id = 群自身 space_id，非空；原 is_external / source_space_name 不受影响（YUJ-63）。
func TestSyncMessages_InternalMember_HomeSpace(t *testing.T) {
	markers := map[string]group.MemberExternalMarker{
		"internal-uid": {
			IsExternal:    0,
			HomeSpaceID:   "space_group_owner",
			HomeSpaceName: "GroupOwnerSpace",
		},
	}
	messages := []*MsgSyncResp{{FromUID: "internal-uid"}}

	applyExternalMarkers(messages, markers)

	// 原语义保留：内部成员不是外部、不带 source_space_name
	assert.Equal(t, 0, messages[0].FromIsExternal)
	assert.Equal(t, "", messages[0].FromSourceSpaceName)
	// 新字段：内部成员的 home 等于群自身 space_id
	assert.Equal(t, "space_group_owner", messages[0].FromHomeSpaceID,
		"内部成员 from_home_space_id 应等于 group.space_id")
	assert.Equal(t, "GroupOwnerSpace", messages[0].FromHomeSpaceName,
		"内部成员 from_home_space_name 应等于群自身 Space 名称")
}

// TestSyncMessages_UnknownSender_HomeSpace 已退群 / 非群成员发送者：
// home_space_* 默认留空，不泄漏其他群的归属信息。
func TestSyncMessages_UnknownSender_HomeSpace(t *testing.T) {
	markers := map[string]group.MemberExternalMarker{
		"ext-uid": {
			IsExternal:      1,
			SourceSpaceName: "ExampleCorp",
			HomeSpaceID:     "space_example",
			HomeSpaceName:   "ExampleCorp",
		},
	}
	messages := []*MsgSyncResp{{FromUID: "ghost-uid"}}

	applyExternalMarkers(messages, markers)

	assert.Equal(t, 0, messages[0].FromIsExternal)
	assert.Equal(t, "", messages[0].FromSourceSpaceName)
	assert.Equal(t, "", messages[0].FromHomeSpaceID,
		"已退群/非成员不应被填 home_space_id")
	assert.Equal(t, "", messages[0].FromHomeSpaceName)
}

// TestMergeforward_HomeSpaceOnUserItems mergeforward payload.users 内每个元素都应带上
// home_space_id / home_space_name（外部 → 来源 space；内部 → 群 space；未知 → 空）。
func TestMergeforward_HomeSpaceOnUserItems(t *testing.T) {
	rawPayload := `{
		"type": 11,
		"title": "聊天记录",
		"users": [
			{"uid": "ext-uid", "name": "外部张三"},
			{"uid": "internal-uid", "name": "内部李四"},
			{"uid": "ghost-uid", "name": "已退群王五"}
		],
		"messages": [{"content": "hello"}]
	}`
	payload := decodePayload(t, rawPayload)

	markers := map[string]group.MemberExternalMarker{
		"ext-uid": {
			IsExternal:      1,
			SourceSpaceName: "ExampleCorp",
			HomeSpaceID:     "space_example",
			HomeSpaceName:   "ExampleCorp",
		},
		"internal-uid": {
			IsExternal:    0,
			HomeSpaceID:   "space_group_owner",
			HomeSpaceName: "GroupOwnerSpace",
		},
	}
	messages := []*MsgSyncResp{{FromUID: "ext-uid", Payload: payload}}

	applyExternalMarkers(messages, markers)

	users, _ := payload["users"].([]interface{})
	assert.Len(t, users, 3)

	u0 := users[0].(map[string]interface{})
	assert.Equal(t, 1, u0["is_external"])
	assert.Equal(t, "ExampleCorp", u0["source_space_name"])
	assert.Equal(t, "space_example", u0["home_space_id"])
	assert.Equal(t, "ExampleCorp", u0["home_space_name"])

	u1 := users[1].(map[string]interface{})
	assert.Equal(t, 0, u1["is_external"])
	assert.Equal(t, "", u1["source_space_name"])
	assert.Equal(t, "space_group_owner", u1["home_space_id"],
		"mergeforward 中内部成员的 home_space_id 应为群 space_id")
	assert.Equal(t, "GroupOwnerSpace", u1["home_space_name"])

	u2 := users[2].(map[string]interface{})
	assert.Equal(t, 0, u2["is_external"])
	assert.Equal(t, "", u2["source_space_name"])
	assert.Equal(t, "", u2["home_space_id"],
		"已退群用户 home_space_id 应为空，避免跨群泄漏")
	assert.Equal(t, "", u2["home_space_name"])
}
