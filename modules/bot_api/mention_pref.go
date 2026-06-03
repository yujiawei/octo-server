package bot_api

import (
	"errors"

	"github.com/Mininglamp-OSS/octo-lib/pkg/wkhttp"
	"github.com/gocraft/dbr/v2"
	"go.uber.org/zap"
)

// getMentionPref handles GET /v1/bot/groups/:group_no/mention_pref.
//
// Adapter-facing read of the per-group no-@ preference (octo-server#237).
// robot_id is taken from the authBot context — query is NOT trusted.
// Membership gate mirrors getGroupInfo (groups.go:60-94). No record → no_mention=0.
func (ba *BotAPI) getMentionPref(c *wkhttp.Context) {
	robotID := getRobotIDFromContext(c)
	if robotID == "" {
		c.ResponseError(errors.New("robot_id not found"))
		return
	}
	groupNo := c.Param("group_no")

	var count int
	err := ba.db.session.SelectBySql(
		"SELECT COUNT(*) FROM group_member WHERE group_no=? AND uid=? AND is_deleted=0",
		groupNo, robotID,
	).LoadOne(&count)
	if err != nil || count == 0 {
		c.ResponseError(errors.New("bot is not a member of this group"))
		return
	}

	var noMention int
	err = ba.db.session.Select("no_mention").From("bot_mention_pref").
		Where("robot_id=? AND group_no=?", robotID, groupNo).LoadOne(&noMention)
	if err != nil && !errors.Is(err, dbr.ErrNotFound) {
		ba.Error("查询 bot_mention_pref 失败", zap.Error(err),
			zap.String("robot_id", robotID), zap.String("group_no", groupNo))
		c.ResponseError(errors.New("查询失败"))
		return
	}

	c.Response(map[string]interface{}{"no_mention": noMention})
}
