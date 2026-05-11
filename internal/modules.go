// YUJ-438 overlay workaround (octo-release, 2026-05-11).
//
// This file is an OSS-only overlay that ONLY affects the published
// Mininglamp-OSS/octo-server mirror — the internal dmwork-org/dmworkim
// source tree is NOT touched.
//
// Why it exists:
//   `docker-compose up` on a fresh OSS install runs the built-in gorp
//   migrations in module-import order. The migration
//   `botfather-20260417-01.sql` does `ALTER TABLE robot ...`, but the
//   `robot` table is created by the `modules/robot` module's migrations.
//   The original import order (`botfather` at position 3, `robot` at
//   position 14) therefore panics octo-server on first boot for OSS
//   users.
//
// The fix here re-orders the two imports so that `robot` runs before
// `botfather`. The internal source keeps the original order because the
// internal production MySQL already has the `robot` table seeded by an
// earlier snapshot; only OSS first-time installers hit the ordering
// bug. Keeping the fix in the tool layer avoids a noisy patch to
// `dmwork-org/dmworkim`.
//
// Ref: YUJ-438 ("OCTO octo-temp 工具层紧急修复 — migration fix +
//       MASTER_KEY 文档 + gitlab-ci 拉黑"), Yu 2026-05-11.

package modules

// 引入模块
import (
	_ "github.com/Mininglamp-OSS/octo-server/modules/backup"
	_ "github.com/Mininglamp-OSS/octo-server/modules/base"
	// YUJ-438: `robot` must import BEFORE `botfather` so that the
	// `robot` table exists by the time `botfather-20260417-01.sql`
	// runs its `ALTER TABLE robot ...`.
	_ "github.com/Mininglamp-OSS/octo-server/modules/robot"
	_ "github.com/Mininglamp-OSS/octo-server/modules/botfather"
	_ "github.com/Mininglamp-OSS/octo-server/modules/category"
	_ "github.com/Mininglamp-OSS/octo-server/modules/channel"
	_ "github.com/Mininglamp-OSS/octo-server/modules/common"
	_ "github.com/Mininglamp-OSS/octo-server/modules/file"
	_ "github.com/Mininglamp-OSS/octo-server/modules/group"
	_ "github.com/Mininglamp-OSS/octo-server/modules/message"
	_ "github.com/Mininglamp-OSS/octo-server/modules/openapi"
	_ "github.com/Mininglamp-OSS/octo-server/modules/qrcode"
	_ "github.com/Mininglamp-OSS/octo-server/modules/report"
	_ "github.com/Mininglamp-OSS/octo-server/modules/search"
	_ "github.com/Mininglamp-OSS/octo-server/modules/space"
	_ "github.com/Mininglamp-OSS/octo-server/modules/statistics"
	_ "github.com/Mininglamp-OSS/octo-server/modules/thread"
	_ "github.com/Mininglamp-OSS/octo-server/modules/user"
	_ "github.com/Mininglamp-OSS/octo-server/modules/voice"
	_ "github.com/Mininglamp-OSS/octo-server/modules/webhook"
	_ "github.com/Mininglamp-OSS/octo-server/modules/workplace"
)
