package message

import (
	"github.com/Mininglamp-OSS/octo-lib/common"
	"github.com/Mininglamp-OSS/octo-lib/config"
	"github.com/Mininglamp-OSS/octo-lib/pkg/log"
	"go.uber.org/zap"
)

// fillPersonSpaceUnread 为 Person 频道计算 per-Space 未读计数。
// 仅处理 channelType=1 且 unread>0 的会话。
// 通过解析消息 payload 中的 space_id 字段来统计属于指定 Space 的未读消息数。
func fillPersonSpaceUnread(
	conversations []*SyncUserConversationResp,
	rawConversations []*config.SyncUserConversationResp,
	spaceID string,
	loginUID string,
	ctx *config.Context,
) {
	if spaceID == "" || len(conversations) == 0 {
		return
	}

	// channelID -> raw conversation
	rawMap := make(map[string]*config.SyncUserConversationResp, len(rawConversations))
	for _, raw := range rawConversations {
		rawMap[raw.ChannelID] = raw
	}

	for _, conv := range conversations {
		if conv.ChannelType != common.ChannelTypePerson.Uint8() || conv.Unread <= 0 {
			continue
		}

		raw := rawMap[conv.ChannelID]
		if raw == nil {
			continue
		}

		readSeq := raw.LastMsgSeq - int64(raw.Unread)

		var messages []*config.MessageResp

		if len(raw.Recents) >= raw.Unread {
			// Recents 覆盖了所有未读消息，直接使用
			messages = raw.Recents
		} else {
			// Recents 不足，从 WuKongIM 拉取未读消息
			startSeq := readSeq + 1
			if startSeq < 1 {
				startSeq = 1
			}
			resp, err := ctx.IMSyncChannelMessage(config.SyncChannelMessageReq{
				LoginUID:        loginUID,
				ChannelID:       conv.ChannelID,
				ChannelType:     conv.ChannelType,
				StartMessageSeq: uint32(startSeq),
				EndMessageSeq:   uint32(raw.LastMsgSeq),
				Limit:           raw.Unread,
				PullMode:        config.PullModeDown,
			})
			if err != nil {
				log.Warn("获取Person未读消息失败，跳过space_unread",
					zap.Error(err),
					zap.String("channelID", conv.ChannelID),
					zap.String("loginUID", loginUID))
				continue
			}
			messages = resp.Messages
		}

		count := countSpaceUnreadFromMessages(messages, spaceID, readSeq)
		conv.SpaceUnread = &count
	}
}

// countSpaceUnreadFromMessages 遍历消息列表，统计 seq > readSeq 且 payload.space_id == spaceID 的消息数。
func countSpaceUnreadFromMessages(messages []*config.MessageResp, spaceID string, readSeq int64) int {
	count := 0
	for _, msg := range messages {
		if int64(msg.MessageSeq) <= readSeq {
			continue
		}
		payloadMap, err := msg.GetPayloadMap()
		if err != nil || payloadMap == nil {
			continue
		}
		if sid, ok := payloadMap["space_id"].(string); ok && sid == spaceID {
			count++
		}
	}
	return count
}
