package botfather

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Mininglamp-OSS/octo-lib/common"
	"github.com/Mininglamp-OSS/octo-lib/config"
	"github.com/Mininglamp-OSS/octo-lib/pkg/wkhttp"
	"go.uber.org/zap"
)

// syncMessages 同步频道历史消息
func (bf *BotFather) syncMessages(c *wkhttp.Context) {
	var req BotSyncMessagesReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("数据格式有误"))
		return
	}
	if strings.TrimSpace(req.ChannelID) == "" {
		c.ResponseError(errors.New("channel_id不能为空"))
		return
	}
	if req.ChannelType == 0 {
		c.ResponseError(errors.New("channel_type不能为空"))
		return
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 200 {
		req.Limit = 200
	}

	robotID := getRobotIDFromContext(c)

	// 群聊场景：验证 bot 是否在群内
	if req.ChannelType == common.ChannelTypeGroup.Uint8() {
		var count int
		_, err := bf.db.session.SelectBySql(
			"SELECT COUNT(*) FROM group_member WHERE group_no=? AND uid=? AND is_deleted=0",
			req.ChannelID, robotID,
		).Load(&count)
		if err != nil {
			bf.Error("查询群成员失败", zap.Error(err))
			c.ResponseError(errors.New("查询群成员失败"))
			return
		}
		if count == 0 {
			c.ResponseError(errors.New("bot is not a member of this group"))
			return
		}
	}

	syncReq := config.SyncChannelMessageReq{
		LoginUID:        robotID,
		ChannelID:       req.ChannelID,
		ChannelType:     req.ChannelType,
		StartMessageSeq: req.StartMessageSeq,
		EndMessageSeq:   req.EndMessageSeq,
		Limit:           req.Limit,
		PullMode:        config.PullMode(req.PullMode),
	}
	resp, err := bf.ctx.IMSyncChannelMessage(syncReq)
	if err != nil {
		bf.Error("同步消息失败", zap.Error(err))
		c.ResponseError(errors.New("同步消息失败"))
		return
	}

	c.Response(resp)
}

// getGroups 获取机器人所在的群组列表
func (bf *BotFather) getGroups(c *wkhttp.Context) {
	robotID := getRobotIDFromContext(c)
	if robotID == "" {
		c.ResponseError(errors.New("robot_id not found"))
		return
	}

	type GroupInfo struct {
		GroupNo string `json:"group_no"`
		Name    string `json:"name"`
	}

	var groups []GroupInfo
	_, err := bf.ctx.DB().SelectBySql(
		"SELECT gm.group_no, g.name FROM group_member gm INNER JOIN `group` g ON gm.group_no = g.group_no WHERE gm.uid = ? AND gm.is_deleted = 0",
		robotID,
	).Load(&groups)
	if err != nil {
		bf.Error("查询机器人群组失败", zap.Error(err))
		c.ResponseError(errors.New("查询群组失败"))
		return
	}

	c.JSON(http.StatusOK, groups)
}

// getGroupInfo 获取群信息
func (bf *BotFather) getGroupInfo(c *wkhttp.Context) {
	robotID := c.GetString("robot_id")
	groupNo := c.Param("group_no")

	// Verify bot is a member of this group
	var count int
	_, err := bf.db.session.SelectBySql("SELECT COUNT(*) FROM group_member WHERE group_no=? AND uid=? AND is_deleted=0", groupNo, robotID).Load(&count)
	if err != nil || count == 0 {
		c.ResponseError(errors.New("bot is not a member of this group"))
		return
	}

	var group struct {
		GroupNo   string `db:"group_no"`
		Name      string `db:"name"`
		Notice    string `db:"notice"`
		Creator   string `db:"creator"`
		Status    int    `db:"status"`
		CreatedAt string `db:"created_at"`
	}
	_, err = bf.db.session.Select("group_no, name, IFNULL(notice,'') notice, IFNULL(creator,'') creator, status, created_at").
		From("`group`").Where("group_no=?", groupNo).Load(&group)
	if err != nil {
		c.ResponseError(errors.New("group not found"))
		return
	}

	c.Response(map[string]interface{}{
		"group_no":   group.GroupNo,
		"name":       group.Name,
		"notice":     group.Notice,
		"creator":    group.Creator,
		"status":     group.Status,
		"created_at": group.CreatedAt,
	})
}

// getGroupMembers 获取群成员列表
func (bf *BotFather) getGroupMembers(c *wkhttp.Context) {
	robotID := c.GetString("robot_id")
	groupNo := c.Param("group_no")

	// Verify bot is a member
	var count int
	_, err := bf.db.session.SelectBySql("SELECT COUNT(*) FROM group_member WHERE group_no=? AND uid=? AND is_deleted=0", groupNo, robotID).Load(&count)
	if err != nil || count == 0 {
		c.ResponseError(errors.New("bot is not a member of this group"))
		return
	}

	type member struct {
		UID       string `db:"uid" json:"uid"`
		Name      string `db:"name" json:"name"`
		Role      int    `db:"role" json:"role"`
		Robot     int    `db:"robot" json:"robot"`
		CreatedAt string `db:"created_at" json:"created_at"`
	}

	var members []member
	_, err = bf.db.session.SelectBySql(`
		SELECT gm.uid, IFNULL(u.name,'') name, gm.role, IFNULL(u.robot,0) robot, gm.created_at 
		FROM group_member gm 
		LEFT JOIN user u ON gm.uid = u.uid 
		WHERE gm.group_no = ? AND gm.is_deleted = 0
		ORDER BY gm.role DESC, gm.created_at ASC
	`, groupNo).Load(&members)
	if err != nil {
		c.ResponseError(err)
		return
	}

	c.Response(members)
}
