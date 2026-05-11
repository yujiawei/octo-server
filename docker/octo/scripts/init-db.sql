-- -----------------------------------------------------------------------------
-- OCTO Server · MySQL initial bootstrap (schema snapshot mode)
-- -----------------------------------------------------------------------------
-- Loaded ONCE on first MySQL container start via docker-entrypoint-initdb.d.
-- Seeds the database with a production-validated schema AND a pre-populated
-- gorp_migrations table so the Go binary's boot-time migration runner finds
-- everything already applied and proceeds straight to serving traffic.
--
-- Why snapshot instead of letting `sql-migrate` do its thing? The internal
-- SQL files have cross-module dependencies (e.g. botfather-*.sql ALTERs the
-- robot table owned by the robot module, category-*.sql UPDATEs
-- group_setting owned by the group module). Those dependencies happen to
-- work in internal environments because migrations were applied
-- incrementally over three years. On a clean docker-compose install the
-- planner sees the full set and orders strictly by filename, producing a
-- schedule that the schema does not support. See YUJ-440.
--
-- Conditional modules:
--   * thread-* migrations are NOT seeded here. The thread module only
--     registers its SQLDir when DM_THREAD_ON=true, so including thread-*
--     ids in gorp_migrations would make sql-migrate complain about
--     "unknown migration in database" on default OSS installs. When the
--     operator opts in by setting DM_THREAD_ON=true, the migration runner
--     will apply those SQLs fresh.
--
-- Refresh procedure: dump from a healthy internal environment and re-run
-- tools/octo-release/scripts/build-init-db.sh. Schema snapshots are
-- versioned with each OCTO release.
--
-- Generated: 2026-05-11 19:31:43Z
-- -----------------------------------------------------------------------------

SET FOREIGN_KEY_CHECKS = 0;
SET UNIQUE_CHECKS = 0;
SET NAMES utf8mb4;

-- ============================================================
-- Schema (81 tables)
-- ============================================================


DROP TABLE IF EXISTS `app`;
CREATE TABLE `app` (
  `app_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'app id',
  `app_key` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'app key',
  `status` int NOT NULL DEFAULT '0' COMMENT '状态 0.禁用 1.可用',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `app_name` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'app名字',
  `app_logo` varchar(400) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'app logo',
  UNIQUE KEY `app_id` (`app_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `app_bot`
--

DROP TABLE IF EXISTS `app_bot`;
CREATE TABLE `app_bot` (
  `id` varchar(40) COLLATE utf8mb4_general_ci NOT NULL,
  `uid` varchar(40) COLLATE utf8mb4_general_ci NOT NULL,
  `display_name` varchar(100) COLLATE utf8mb4_general_ci NOT NULL,
  `description` varchar(500) COLLATE utf8mb4_general_ci DEFAULT '',
  `avatar` varchar(200) COLLATE utf8mb4_general_ci DEFAULT '',
  `scope` varchar(20) COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'platform' COMMENT 'platform or space',
  `space_id` varchar(40) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '0=draft 1=published 2=unpublished',
  `token` varchar(100) COLLATE utf8mb4_general_ci NOT NULL,
  `welcome_msg` varchar(500) COLLATE utf8mb4_general_ci DEFAULT '',
  `created_by` varchar(40) COLLATE utf8mb4_general_ci NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid` (`uid`),
  UNIQUE KEY `token` (`token`),
  KEY `idx_scope_status` (`scope`,`status`),
  KEY `idx_space_status` (`space_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `app_config`
--

DROP TABLE IF EXISTS `app_config`;
CREATE TABLE `app_config` (
  `id` int NOT NULL AUTO_INCREMENT,
  `rsa_private_key` varchar(4000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `rsa_public_key` varchar(4000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `version` int NOT NULL DEFAULT '0',
  `super_token` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `super_token_on` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `revoke_second` smallint NOT NULL DEFAULT '0' COMMENT '消息可撤回时长',
  `welcome_message` varchar(2000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '登录欢迎语',
  `new_user_join_system_group` smallint NOT NULL DEFAULT '1' COMMENT '注册用户是否默认加入系统群',
  `search_by_phone` smallint NOT NULL DEFAULT '0' COMMENT '是否可通过手机号搜索',
  `register_invite_on` smallint NOT NULL DEFAULT '0' COMMENT '是否开启注册邀请',
  `send_welcome_message_on` smallint NOT NULL DEFAULT '1' COMMENT '是否开启登录欢迎语',
  `invite_system_account_join_group_on` smallint NOT NULL DEFAULT '0' COMMENT '是否开启系统账号进入群聊',
  `register_user_must_complete_info_on` smallint NOT NULL DEFAULT '0' COMMENT '注册用户是否必须完善信息',
  `channel_pinned_message_max_count` smallint NOT NULL DEFAULT '10' COMMENT '频道最多置顶消息数量',
  `can_modify_api_url` smallint NOT NULL DEFAULT '0' COMMENT '是否能修改服务器地址',
  `destroy_cooling_off_days` int NOT NULL DEFAULT '7' COMMENT '注销冷静期天数',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `app_module`
--

DROP TABLE IF EXISTS `app_module`;
CREATE TABLE `app_module` (
  `id` int NOT NULL AUTO_INCREMENT,
  `sid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `name` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `desc` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `status` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `app_module_sid_idx` (`sid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `app_version`
--

DROP TABLE IF EXISTS `app_version`;
CREATE TABLE `app_version` (
  `id` int NOT NULL AUTO_INCREMENT,
  `app_version` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `os` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `is_force` smallint NOT NULL DEFAULT '0',
  `update_desc` text COLLATE utf8mb4_general_ci NOT NULL COMMENT '更新说明',
  `download_url` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `signature` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '二进制包的签名',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `backup_config`
--

DROP TABLE IF EXISTS `backup_config`;
CREATE TABLE `backup_config` (
  `id` int NOT NULL AUTO_INCREMENT,
  `enabled` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否启用备份',
  `prefix` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'backup/' COMMENT '备份路径前缀',
  `cron_expr` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '0 2 * * *' COMMENT 'cron表达式',
  `retention_count` int NOT NULL DEFAULT '7' COMMENT '保留备份数量',
  `data_dir` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '/data/wukongim' COMMENT 'WuKongIM数据目录',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='备份配置表';

--
-- Table structure for table `backup_history`
--

DROP TABLE IF EXISTS `backup_history`;
CREATE TABLE `backup_history` (
  `id` int NOT NULL AUTO_INCREMENT,
  `backup_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '备份ID (UUID)',
  `status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending' COMMENT '状态: pending/running/success/failed',
  `file_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '备份文件名',
  `file_size` bigint NOT NULL DEFAULT '0' COMMENT '文件大小 (bytes)',
  `storage_path` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '存储路径',
  `started_at` datetime DEFAULT NULL COMMENT '开始时间',
  `finished_at` datetime DEFAULT NULL COMMENT '完成时间',
  `error_message` text COLLATE utf8mb4_unicode_ci COMMENT '错误信息',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_backup_id` (`backup_id`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='备份历史表';

--
-- Table structure for table `channel_offset`
--

DROP TABLE IF EXISTS `channel_offset`;
CREATE TABLE `channel_offset` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid_channel_idx` (`uid`,`channel_id`,`channel_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `channel_offset1`
--

DROP TABLE IF EXISTS `channel_offset1`;
CREATE TABLE `channel_offset1` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid_channel_idx` (`uid`,`channel_id`,`channel_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `channel_offset2`
--

DROP TABLE IF EXISTS `channel_offset2`;
CREATE TABLE `channel_offset2` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid_channel_idx` (`uid`,`channel_id`,`channel_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `channel_setting`
--

DROP TABLE IF EXISTS `channel_setting`;
CREATE TABLE `channel_setting` (
  `id` int NOT NULL AUTO_INCREMENT,
  `channel_id` varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `channel_type` smallint NOT NULL DEFAULT '0',
  `parent_channel_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `parent_channel_type` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `msg_auto_delete` int NOT NULL DEFAULT '0' COMMENT '消息定时删除时间',
  `offset_message_seq` int NOT NULL DEFAULT '0' COMMENT 'channel消息删除偏移seq',
  PRIMARY KEY (`id`),
  UNIQUE KEY `channel_setting_uidx` (`channel_id`,`channel_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `chat_bg`
--

DROP TABLE IF EXISTS `chat_bg`;
CREATE TABLE `chat_bg` (
  `id` int NOT NULL AUTO_INCREMENT,
  `cover` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `url` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `is_svg` smallint NOT NULL DEFAULT '1',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `conversation_extra`
--

DROP TABLE IF EXISTS `conversation_extra`;
CREATE TABLE `conversation_extra` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '所属用户',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '频道ID',
  `channel_type` smallint NOT NULL DEFAULT '0' COMMENT '频道类型',
  `browse_to` bigint NOT NULL DEFAULT '0' COMMENT '预览到的位置，与会话保持位置不同的是 预览到的位置是用户读到的最大的messageSeq。跟未读消息数量有关系',
  `keep_message_seq` bigint NOT NULL DEFAULT '0' COMMENT '会话保持的位置',
  `keep_offset_y` int NOT NULL DEFAULT '0' COMMENT '会话保持的位置的偏移量',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '更新时间',
  `draft` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '草稿',
  `version` bigint NOT NULL DEFAULT '0' COMMENT '数据版本',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid_channel_idx` (`uid`,`channel_id`,`channel_type`),
  KEY `uid_idx` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `device`
--

DROP TABLE IF EXISTS `device`;
CREATE TABLE `device` (
  `id` int NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `device_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `device_name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `device_model` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `last_login` int NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `device_uid_device_id` (`uid`,`device_id`),
  KEY `device_uid` (`uid`),
  KEY `device_device_id` (`device_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `device_flag`
--

DROP TABLE IF EXISTS `device_flag`;
CREATE TABLE `device_flag` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `device_flag` smallint NOT NULL DEFAULT '0' COMMENT '设备标记 0. app 1.Web 2.PC',
  `weight` int NOT NULL DEFAULT '0' COMMENT '设备权重 值越大越优先',
  `remark` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '备注',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `udx_device_flag` (`device_flag`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `device_offset`
--

DROP TABLE IF EXISTS `device_offset`;
CREATE TABLE `device_offset` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `device_uuid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid_device_offset_unidx` (`uid`,`device_uuid`,`channel_id`,`channel_type`),
  KEY `uid_device_offset_idx` (`uid`,`device_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `event`
--

DROP TABLE IF EXISTS `event`;
CREATE TABLE `event` (
  `id` int NOT NULL AUTO_INCREMENT,
  `event` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `type` smallint NOT NULL DEFAULT '0',
  `data` varchar(10000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `status` smallint NOT NULL DEFAULT '0',
  `reason` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `version_lock` int NOT NULL DEFAULT '0' COMMENT '乐观锁',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `event_key` (`event`),
  KEY `event_type` (`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `friend`
--

DROP TABLE IF EXISTS `friend`;
CREATE TABLE `friend` (
  `id` int NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户UID',
  `to_uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '好友uid',
  `remark` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '对好友的备注 TODO: 此字段不再使用，已经迁移到user_setting表',
  `flag` smallint NOT NULL DEFAULT '0' COMMENT '好友标示',
  `version` bigint NOT NULL DEFAULT '0' COMMENT '版本号',
  `vercode` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '验证码 加好友来源',
  `source_vercode` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '好友来源',
  `is_deleted` smallint NOT NULL DEFAULT '0' COMMENT '是否已删除',
  `is_alone` smallint NOT NULL DEFAULT '0' COMMENT '单项好友',
  `initiator` smallint NOT NULL DEFAULT '0' COMMENT '加好友发起方',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `to_uid_uid` (`uid`,`to_uid`),
  KEY `idx_friend_vercode` (`vercode`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `friend_apply_record`
--

DROP TABLE IF EXISTS `friend_apply_record`;
CREATE TABLE `friend_apply_record` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `to_uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `remark` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `status` smallint NOT NULL DEFAULT '1',
  `token` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `friend_apply_record_uid_touidx` (`uid`,`to_uid`),
  KEY `friend_apply_record_uidx` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `gitee_user`
--

DROP TABLE IF EXISTS `gitee_user`;
CREATE TABLE `gitee_user` (
  `id` bigint NOT NULL DEFAULT '0' COMMENT '用户 ID',
  `login` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户名',
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户姓名',
  `email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户邮箱',
  `bio` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户简介',
  `avatar_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户头像 URL',
  `blog` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户博客 URL',
  `events_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户事件 URL',
  `followers` int NOT NULL DEFAULT '0' COMMENT '用户粉丝数',
  `followers_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户粉丝 URL',
  `following` int NOT NULL DEFAULT '0' COMMENT '用户关注数',
  `following_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户关注 URL',
  `gists_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户 Gist URL',
  `html_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户主页 URL',
  `member_role` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户角色',
  `organizations_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户组织 URL',
  `public_gists` int NOT NULL DEFAULT '0' COMMENT '用户公开 Gist 数',
  `public_repos` int NOT NULL DEFAULT '0' COMMENT '用户公开仓库数',
  `received_events_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户接收事件 URL',
  `remark` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '企业备注名',
  `repos_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户仓库 URL',
  `stared` int NOT NULL DEFAULT '0' COMMENT '用户收藏数',
  `starred_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户收藏 URL',
  `subscriptions_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户订阅 URL',
  `url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户 URL',
  `watched` int NOT NULL DEFAULT '0' COMMENT '用户关注的仓库数',
  `weibo` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户微博 URL',
  `type` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户类型',
  `gitee_created_at` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'gitee用户创建时间',
  `gitee_updated_at` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'gitee用户更新时间',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `gitee_user_login` (`login`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `github_user`
--

DROP TABLE IF EXISTS `github_user`;
CREATE TABLE `github_user` (
  `id` bigint NOT NULL DEFAULT '0' COMMENT '用户 ID',
  `login` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '登录名',
  `node_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '节点ID',
  `avatar_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '头像URL',
  `gravatar_id` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'Gravatar ID',
  `url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'GitHub URL',
  `html_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'GitHub HTML URL',
  `followers_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '关注者URL',
  `following_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '被关注者URL',
  `gists_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '代码片段URL',
  `starred_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '收藏URL',
  `subscriptions_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '订阅URL',
  `organizations_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '组织URL',
  `repos_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '仓库URL',
  `events_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '事件URL',
  `received_events_url` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '接收事件URL',
  `type` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '用户类型',
  `site_admin` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否为管理员',
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '姓名',
  `company` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '公司',
  `blog` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '博客',
  `location` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '所在地',
  `email` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '电子邮件',
  `hireable` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否可被雇佣',
  `bio` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '个人简介',
  `twitter_username` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'Twitter 用户名',
  `public_repos` int NOT NULL DEFAULT '0' COMMENT '公共仓库数量',
  `public_gists` int NOT NULL DEFAULT '0' COMMENT '公共代码片段数量',
  `followers` int NOT NULL DEFAULT '0' COMMENT '关注者数量',
  `following` int NOT NULL DEFAULT '0' COMMENT '被关注者数量',
  `github_created_at` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建时间',
  `github_updated_at` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '更新时间',
  `private_gists` int NOT NULL DEFAULT '0' COMMENT '私有代码片段数量',
  `total_private_repos` int NOT NULL DEFAULT '0' COMMENT '私有仓库总数',
  `owned_private_repos` int NOT NULL DEFAULT '0' COMMENT '拥有的私有仓库数量',
  `disk_usage` int NOT NULL DEFAULT '0' COMMENT '磁盘使用量',
  `collaborators` int NOT NULL DEFAULT '0' COMMENT '协作者数量',
  `two_factor_authentication` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否启用两步验证',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `github_user_login` (`login`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `gorp_migrations`
--

CREATE TABLE `gorp_migrations` (
  `id` varchar(255) NOT NULL,
  `applied_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

--
-- Table structure for table `group`
--

DROP TABLE IF EXISTS `group`;
CREATE TABLE `group` (
  `id` int NOT NULL AUTO_INCREMENT,
  `group_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `name` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `creator` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `status` smallint NOT NULL DEFAULT '0',
  `forbidden` smallint NOT NULL DEFAULT '0' COMMENT '群禁言',
  `invite` smallint NOT NULL DEFAULT '0' COMMENT '群邀请开关',
  `forbidden_add_friend` smallint NOT NULL DEFAULT '0',
  `allow_view_history_msg` smallint NOT NULL DEFAULT '1',
  `version` bigint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `notice` varchar(400) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `avatar` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '群头像',
  `is_upload_avatar` smallint NOT NULL DEFAULT '0' COMMENT '群头像是否已经被用户上传',
  `group_type` smallint NOT NULL DEFAULT '0' COMMENT '群类型 0.普通群 1.超大群',
  `category` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '群分类',
  `allow_member_pinned_message` smallint NOT NULL DEFAULT '0' COMMENT '允许成员置顶聊天消息 0.不允许 1.允许',
  `space_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '' COMMENT 'Space ID',
  `group_md` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'GROUP.md content',
  `group_md_version` bigint NOT NULL DEFAULT '0' COMMENT 'GROUP.md version (auto-increment on update)',
  `group_md_updated_at` timestamp NULL DEFAULT NULL COMMENT 'GROUP.md last update time',
  `group_md_updated_by` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'GROUP.md last updater UID',
  `is_external_group` smallint NOT NULL DEFAULT '0' COMMENT 'External group: 0=no, 1=yes (auto-maintained when external members join/leave)',
  `allow_external` smallint NOT NULL DEFAULT '1' COMMENT 'Allow external members: 1=yes (default, backward-compat), 0=block external scan-join and invite',
  PRIMARY KEY (`id`),
  UNIQUE KEY `group_groupNo` (`group_no`),
  KEY `group_creator` (`creator`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `group_category`
--

DROP TABLE IF EXISTS `group_category`;
CREATE TABLE `group_category` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `category_id` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '类别ID',
  `space_id` varchar(40) COLLATE utf8mb4_general_ci NOT NULL COMMENT '所属空间',
  `uid` varchar(40) COLLATE utf8mb4_general_ci NOT NULL COMMENT '拥有者',
  `name` varchar(100) COLLATE utf8mb4_general_ci NOT NULL COMMENT '类别名称',
  `sort` int NOT NULL DEFAULT '0' COMMENT '排序权重（越小越靠前）',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '1=正常 2=已删除',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `is_default` tinyint DEFAULT NULL COMMENT '1=默认未分类分组（不可删除/重命名），NULL=普通分组',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_category_id` (`category_id`),
  UNIQUE KEY `uk_uid_space_is_default` (`uid`,`space_id`,`is_default`),
  KEY `idx_uid_space_sort` (`uid`,`space_id`,`sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='群组类别表（用户个人视图）';

--
-- Table structure for table `group_invite`
--

DROP TABLE IF EXISTS `group_invite`;
CREATE TABLE `group_invite` (
  `id` int NOT NULL AUTO_INCREMENT,
  `invite_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '邀请唯一编号',
  `group_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '群唯一编号',
  `inviter` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '邀请者uid',
  `remark` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '备注',
  `status` smallint NOT NULL DEFAULT '0' COMMENT '状态： 0.待确认 1.已确认',
  `allower` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '允许此次操作的用户uid',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `group_member`
--

DROP TABLE IF EXISTS `group_member`;
CREATE TABLE `group_member` (
  `id` int NOT NULL AUTO_INCREMENT,
  `group_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `remark` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `role` smallint NOT NULL DEFAULT '0',
  `version` bigint NOT NULL DEFAULT '0',
  `is_deleted` smallint NOT NULL DEFAULT '0',
  `status` smallint NOT NULL DEFAULT '1',
  `vercode` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `robot` smallint NOT NULL DEFAULT '0',
  `invite_uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `forbidden_expir_time` int NOT NULL DEFAULT '0' COMMENT '群成员禁言时长',
  `bot_admin` smallint NOT NULL DEFAULT '0' COMMENT 'Bot admin: 0=no, 1=yes',
  `is_external` smallint NOT NULL DEFAULT '0' COMMENT 'External member: 0=no, 1=yes',
  `source_space_id` varchar(40) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'Source Space ID for external members',
  PRIMARY KEY (`id`),
  UNIQUE KEY `group_no_uid` (`group_no`,`uid`),
  KEY `group_member_groupNo` (`group_no`),
  KEY `group_member_uid` (`uid`),
  KEY `idx_group_member_external` (`uid`,`is_external`,`is_deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `group_setting`
--

DROP TABLE IF EXISTS `group_setting`;
CREATE TABLE `group_setting` (
  `id` int NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `group_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `remark` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `mute` smallint NOT NULL DEFAULT '0',
  `top` smallint NOT NULL DEFAULT '0',
  `show_nick` smallint NOT NULL DEFAULT '0',
  `save` smallint NOT NULL DEFAULT '0',
  `chat_pwd_on` smallint NOT NULL DEFAULT '0',
  `revoke_remind` smallint NOT NULL DEFAULT '1',
  `join_group_remind` smallint NOT NULL DEFAULT '0',
  `screenshot` smallint NOT NULL DEFAULT '0',
  `receipt` smallint NOT NULL DEFAULT '0',
  `version` bigint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `flame` smallint NOT NULL DEFAULT '0' COMMENT '阅后即焚是否开启 1.开启 0.未开启',
  `flame_second` smallint NOT NULL DEFAULT '0' COMMENT '阅后即焚销毁秒数',
  `category_id` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '用户自定义类别ID',
  `category_sort` int NOT NULL DEFAULT '0' COMMENT '类别内排序',
  PRIMARY KEY (`id`),
  UNIQUE KEY `groupsetting_group_no_uid` (`group_no`,`uid`),
  KEY `group_setting_groupNo` (`group_no`),
  KEY `group_setting_uid` (`uid`),
  KEY `idx_uid_category` (`uid`,`category_id`,`category_sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `invite_item`
--

DROP TABLE IF EXISTS `invite_item`;
CREATE TABLE `invite_item` (
  `id` int NOT NULL AUTO_INCREMENT,
  `invite_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '邀请唯一编号',
  `group_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '群唯一编号',
  `inviter` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '邀请者uid',
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '被邀请者uid',
  `status` smallint NOT NULL DEFAULT '0' COMMENT '状态： 0.待确认 1.已确认',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `login_log`
--

DROP TABLE IF EXISTS `login_log`;
CREATE TABLE `login_log` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) NOT NULL DEFAULT '' COMMENT '用户OpenId',
  `login_ip` varchar(40) NOT NULL DEFAULT '' COMMENT '最后一次登录ip',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

--
-- Table structure for table `member_readed`
--

DROP TABLE IF EXISTS `member_readed`;
CREATE TABLE `member_readed` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `clone_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `message_id` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `message_uid_idx` (`message_id`,`uid`),
  KEY `channel_idx` (`channel_id`,`channel_type`),
  KEY `uid_idx` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `message`
--

DROP TABLE IF EXISTS `message`;
CREATE TABLE `message` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `message_id` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `client_msg_no` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `header` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `setting` smallint NOT NULL DEFAULT '0',
  `signal` smallint NOT NULL DEFAULT '0',
  `from_uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `timestamp` bigint NOT NULL DEFAULT '0',
  `payload` mediumblob NOT NULL,
  `is_deleted` smallint NOT NULL DEFAULT '0',
  `voice_status` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `expire` int NOT NULL DEFAULT '0' COMMENT '消息过期时长',
  `expire_at` bigint NOT NULL DEFAULT '0' COMMENT '消息过期时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `message_id` (`message_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `message1`
--

DROP TABLE IF EXISTS `message1`;
CREATE TABLE `message1` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `message_id` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `client_msg_no` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `setting` smallint NOT NULL DEFAULT '0',
  `signal` smallint NOT NULL DEFAULT '0',
  `header` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `from_uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `timestamp` bigint NOT NULL DEFAULT '0',
  `payload` mediumblob NOT NULL,
  `is_deleted` smallint NOT NULL DEFAULT '0',
  `voice_status` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `expire` int NOT NULL DEFAULT '0' COMMENT '消息过期时长',
  `expire_at` bigint NOT NULL DEFAULT '0' COMMENT '消息过期时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `message_id` (`message_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `message2`
--

DROP TABLE IF EXISTS `message2`;
CREATE TABLE `message2` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `message_id` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `client_msg_no` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `setting` smallint NOT NULL DEFAULT '0',
  `signal` smallint NOT NULL DEFAULT '0',
  `header` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `from_uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `timestamp` bigint NOT NULL DEFAULT '0',
  `payload` mediumblob NOT NULL,
  `is_deleted` smallint NOT NULL DEFAULT '0',
  `voice_status` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `expire` int NOT NULL DEFAULT '0' COMMENT '消息过期时长',
  `expire_at` bigint NOT NULL DEFAULT '0' COMMENT '消息过期时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `message_id` (`message_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `message3`
--

DROP TABLE IF EXISTS `message3`;
CREATE TABLE `message3` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `message_id` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `client_msg_no` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `setting` smallint NOT NULL DEFAULT '0',
  `signal` smallint NOT NULL DEFAULT '0',
  `header` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `from_uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `timestamp` bigint NOT NULL DEFAULT '0',
  `payload` mediumblob NOT NULL,
  `is_deleted` smallint NOT NULL DEFAULT '0',
  `voice_status` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `expire` int NOT NULL DEFAULT '0' COMMENT '消息过期时长',
  `expire_at` bigint NOT NULL DEFAULT '0' COMMENT '消息过期时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `message_id` (`message_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `message4`
--

DROP TABLE IF EXISTS `message4`;
CREATE TABLE `message4` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `message_id` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `client_msg_no` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `setting` smallint NOT NULL DEFAULT '0',
  `signal` smallint NOT NULL DEFAULT '0',
  `header` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `from_uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `timestamp` bigint NOT NULL DEFAULT '0',
  `payload` mediumblob NOT NULL,
  `is_deleted` smallint NOT NULL DEFAULT '0',
  `voice_status` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `expire` int NOT NULL DEFAULT '0' COMMENT '消息过期时长',
  `expire_at` bigint NOT NULL DEFAULT '0' COMMENT '消息过期时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `message_id` (`message_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `message_extra`
--

DROP TABLE IF EXISTS `message_extra`;
CREATE TABLE `message_extra` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `message_id` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `from_uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `revoke` smallint NOT NULL DEFAULT '0',
  `revoker` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `clone_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `version` bigint NOT NULL DEFAULT '0',
  `readed_count` int NOT NULL DEFAULT '0',
  `is_deleted` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `content_edit` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '编辑后的正文',
  `content_edit_hash` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '编辑正文的hash值，用于重复判断',
  `edited_at` int NOT NULL DEFAULT '0' COMMENT '编辑时间 时间戳（秒）',
  `is_pinned` smallint NOT NULL DEFAULT '0' COMMENT '消息是否置顶',
  PRIMARY KEY (`id`),
  UNIQUE KEY `message_id` (`message_id`),
  KEY `from_uid_idx` (`from_uid`),
  KEY `channel_idx` (`channel_id`,`channel_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `message_user_extra`
--

DROP TABLE IF EXISTS `message_user_extra`;
CREATE TABLE `message_user_extra` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `message_id` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `voice_readed` smallint NOT NULL DEFAULT '0',
  `message_is_deleted` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid_message_idx` (`uid`,`message_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `message_user_extra1`
--

DROP TABLE IF EXISTS `message_user_extra1`;
CREATE TABLE `message_user_extra1` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `message_id` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `voice_readed` smallint NOT NULL DEFAULT '0',
  `message_is_deleted` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid_message_idx` (`uid`,`message_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `message_user_extra2`
--

DROP TABLE IF EXISTS `message_user_extra2`;
CREATE TABLE `message_user_extra2` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `message_id` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `voice_readed` smallint NOT NULL DEFAULT '0',
  `message_is_deleted` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid_message_idx` (`uid`,`message_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `oidc_audit_log`
--

DROP TABLE IF EXISTS `oidc_audit_log`;
CREATE TABLE `oidc_audit_log` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(64) NOT NULL DEFAULT '',
  `event` varchar(32) NOT NULL DEFAULT '',
  `ip` varchar(45) NOT NULL DEFAULT '',
  `user_agent` varchar(512) NOT NULL DEFAULT '',
  `reason` varchar(255) NOT NULL DEFAULT '',
  `trace_id` varchar(64) NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_uid_time` (`uid`,`created_at`),
  KEY `idx_event_time` (`event`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

--
-- Table structure for table `pinned_message`
--

DROP TABLE IF EXISTS `pinned_message`;
CREATE TABLE `pinned_message` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `message_id` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `is_deleted` smallint NOT NULL DEFAULT '0',
  `version` bigint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pinned_message_message_idx` (`message_id`),
  KEY `pinned_message_channelx` (`channel_id`,`channel_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `prohibit_words`
--

DROP TABLE IF EXISTS `prohibit_words`;
CREATE TABLE `prohibit_words` (
  `id` int NOT NULL AUTO_INCREMENT,
  `is_deleted` smallint NOT NULL DEFAULT '0',
  `version` bigint NOT NULL DEFAULT '0',
  `content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `reaction_users`
--

DROP TABLE IF EXISTS `reaction_users`;
CREATE TABLE `reaction_users` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `message_id` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `seq` bigint NOT NULL DEFAULT '0',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `name` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `emoji` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `is_deleted` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `reaction_user_message_channel` (`message_id`,`uid`,`emoji`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `reminder_done`
--

DROP TABLE IF EXISTS `reminder_done`;
CREATE TABLE `reminder_done` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `reminder_id` bigint NOT NULL DEFAULT '0' COMMENT '提醒事项的id',
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '完成的用户uid',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `reminder_done_uid_reminder_id_uidx` (`uid`,`reminder_id`),
  KEY `reminder_done_reminder_id_idx` (`reminder_id`),
  KEY `reminder_done_created_at_idx` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `reminders`
--

DROP TABLE IF EXISTS `reminders`;
CREATE TABLE `reminders` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '频道ID',
  `channel_type` smallint NOT NULL DEFAULT '0' COMMENT '频道类型',
  `reminder_type` int NOT NULL DEFAULT '0' COMMENT '提醒类型 1.有人@我 2.草稿',
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '提醒的用户uid，如果此字段为空则表示 提醒项为整个频道内的成员',
  `text` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '提醒内容',
  `data` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '自定义数据',
  `is_locate` smallint NOT NULL DEFAULT '0' COMMENT ' 是否需要定位',
  `message_seq` bigint NOT NULL DEFAULT '0' COMMENT '消息序列号',
  `message_id` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '消息唯一ID（全局唯一）',
  `version` bigint NOT NULL DEFAULT '0' COMMENT ' 数据版本',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `client_msg_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '消息client msg no',
  `is_deleted` smallint NOT NULL DEFAULT '0' COMMENT '是否被删除',
  `publisher` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '提醒项发布者uid',
  PRIMARY KEY (`id`),
  KEY `channel_uid_uidx` (`uid`,`channel_id`,`channel_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `report`
--

DROP TABLE IF EXISTS `report`;
CREATE TABLE `report` (
  `id` int NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '举报用户',
  `category_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '类别编号',
  `channel_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '频道ID',
  `channel_type` smallint NOT NULL DEFAULT '0' COMMENT '频道类型',
  `imgs` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '图片集合',
  `remark` varchar(800) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '备注',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `report_category`
--

DROP TABLE IF EXISTS `report_category`;
CREATE TABLE `report_category` (
  `id` int NOT NULL AUTO_INCREMENT,
  `category_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '类别编号',
  `category_name` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '类别名称',
  `parent_category_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '父类别编号',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `category_ename` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '英文类别名称',
  PRIMARY KEY (`id`),
  UNIQUE KEY `report_category_no_idx` (`category_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `robot`
--

DROP TABLE IF EXISTS `robot`;
CREATE TABLE `robot` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `robot_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `token` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `version` bigint NOT NULL DEFAULT '0',
  `status` smallint NOT NULL DEFAULT '1',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `inline_on` smallint NOT NULL DEFAULT '0' COMMENT '是否开启行内搜索',
  `placeholder` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '输入框占位符，开启行内搜索有效',
  `username` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '机器人的username',
  `app_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '机器人所属app id',
  `creator_uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建者UID',
  `description` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '机器人描述',
  `bot_token` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'Bot认证Token(bf_前缀)',
  `im_token_cache` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '缓存的IM Token',
  `bot_commands` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '机器人命令列表JSON',
  `auto_approve` tinyint NOT NULL DEFAULT '0' COMMENT '是否自动通过好友申请 0:否 1:是',
  `agent_platform` varchar(50) COLLATE utf8mb4_general_ci DEFAULT '' COMMENT 'AI Agent 平台名称（如 OpenClaw）',
  `agent_version` varchar(50) COLLATE utf8mb4_general_ci DEFAULT '' COMMENT 'Agent 平台版本号（最后一次注册时上报）',
  `plugin_version` varchar(50) COLLATE utf8mb4_general_ci DEFAULT '' COMMENT 'DMWork 插件版本号（最后一次注册时上报）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `robot_id_robot_index` (`robot_id`),
  UNIQUE KEY `idx_robot_bot_token` (`bot_token`),
  KEY `idx_robot_creator_uid` (`creator_uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `robot_apply`
--

DROP TABLE IF EXISTS `robot_apply`;
CREATE TABLE `robot_apply` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '申请人 UID',
  `robot_uid` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Bot UID',
  `owner_uid` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Bot Owner UID',
  `remark` varchar(200) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '申请备注',
  `space_id` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '申请来源 Space',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '0=待处理 1=通过 2=拒绝',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_uid_robot_pending` (`uid`,`robot_uid`,`status`),
  KEY `idx_owner_status` (`owner_uid`,`status`),
  KEY `idx_robot_status` (`robot_uid`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Bot 好友申请记录';

--
-- Table structure for table `robot_menu`
--

DROP TABLE IF EXISTS `robot_menu`;
CREATE TABLE `robot_menu` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `robot_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `cmd` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `remark` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `type` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `bot_id_robot_menu_index` (`robot_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `send_history`
--

DROP TABLE IF EXISTS `send_history`;
CREATE TABLE `send_history` (
  `id` int NOT NULL AUTO_INCREMENT,
  `receiver` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `receiver_name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `receiver_channel_type` smallint NOT NULL DEFAULT '0',
  `sender` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `sender_name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `handler_uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `handler_name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `seq`
--

DROP TABLE IF EXISTS `seq`;
CREATE TABLE `seq` (
  `id` int NOT NULL AUTO_INCREMENT,
  `key` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `min_seq` bigint NOT NULL DEFAULT '1000000',
  `step` int NOT NULL DEFAULT '1000',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `seq_uidx` (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `shortno`
--

DROP TABLE IF EXISTS `shortno`;
CREATE TABLE `shortno` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `shortno` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '唯一短编号',
  `used` smallint NOT NULL DEFAULT '0' COMMENT '是否被用',
  `hold` smallint NOT NULL DEFAULT '0' COMMENT '保留，保留的号码将不会再被分配',
  `locked` smallint NOT NULL DEFAULT '0' COMMENT '是否被锁定，锁定了的短编号将不再被分配,直到解锁',
  `business` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '被使用的业务，比如 user',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `udx_shortno` (`shortno`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `signal_identities`
--

DROP TABLE IF EXISTS `signal_identities`;
CREATE TABLE `signal_identities` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `registration_id` bigint NOT NULL DEFAULT '0',
  `identity_key` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `signed_prekey_id` int NOT NULL DEFAULT '0',
  `signed_pubkey` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `signed_signature` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `identities_index_id` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `signal_onetime_prekeys`
--

DROP TABLE IF EXISTS `signal_onetime_prekeys`;
CREATE TABLE `signal_onetime_prekeys` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `key_id` int NOT NULL DEFAULT '0',
  `pubkey` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `key_id_uid_index_id` (`uid`,`key_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `space`
--

DROP TABLE IF EXISTS `space`;
CREATE TABLE `space` (
  `id` int NOT NULL AUTO_INCREMENT,
  `space_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `description` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `logo` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `creator` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `status` smallint NOT NULL DEFAULT '1',
  `version` bigint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `max_users` int NOT NULL DEFAULT '0' COMMENT '最大成员数 0表示不限制',
  `preset_group_ids` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '预设群组ID列表(JSON数组)，新成员加入Space时自动加入这些群',
  `join_mode` tinyint NOT NULL DEFAULT '0' COMMENT '加入模式 0=直接加入 1=需要审批',
  PRIMARY KEY (`id`),
  UNIQUE KEY `space_spaceid` (`space_id`),
  KEY `space_creator` (`creator`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `space_email_invite`
--

DROP TABLE IF EXISTS `space_email_invite`;
CREATE TABLE `space_email_invite` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `token_hash` varchar(64) NOT NULL COMMENT 'SHA-256 十六进制',
  `invite_type` tinyint NOT NULL COMMENT '1=owner 2=member',
  `email` varchar(200) NOT NULL COMMENT '收件邮箱',
  `space_id` varchar(40) NOT NULL DEFAULT '' COMMENT 'member 类型关联的空间ID',
  `role` tinyint NOT NULL DEFAULT '0' COMMENT 'member 类型角色 0=成员 1=管理员',
  `planned_name` varchar(100) NOT NULL DEFAULT '' COMMENT 'owner 类型计划空间名',
  `planned_description` varchar(500) NOT NULL DEFAULT '',
  `planned_logo` varchar(200) NOT NULL DEFAULT '',
  `planned_max_users` int NOT NULL DEFAULT '0',
  `planned_join_mode` tinyint NOT NULL DEFAULT '0',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '0=pending 1=consumed 2=expired 3=revoked',
  `expires_at` timestamp NULL DEFAULT NULL,
  `created_by` varchar(40) NOT NULL COMMENT '发起人UID',
  `consumed_by` varchar(40) NOT NULL DEFAULT '' COMMENT '接受人UID',
  `consumed_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_token_hash` (`token_hash`),
  KEY `idx_email_status` (`email`,`status`),
  KEY `idx_space_status` (`space_id`,`status`),
  KEY `idx_created_by` (`created_by`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='空间邮件邀请（owner/member）';

--
-- Table structure for table `space_invitation`
--

DROP TABLE IF EXISTS `space_invitation`;
CREATE TABLE `space_invitation` (
  `id` int NOT NULL AUTO_INCREMENT,
  `space_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `invite_code` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `creator` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `max_uses` int NOT NULL DEFAULT '0',
  `used_count` int NOT NULL DEFAULT '0',
  `expires_at` timestamp NULL DEFAULT NULL,
  `status` smallint NOT NULL DEFAULT '1',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `spaceinvite_code` (`invite_code`),
  KEY `spaceinvite_spaceid` (`space_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `space_join_apply`
--

DROP TABLE IF EXISTS `space_join_apply`;
CREATE TABLE `space_join_apply` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `space_id` varchar(40) COLLATE utf8mb4_general_ci NOT NULL COMMENT '空间ID',
  `uid` varchar(40) COLLATE utf8mb4_general_ci NOT NULL COMMENT '申请人UID',
  `invite_code` varchar(20) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '使用的邀请码',
  `remark` varchar(200) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '申请备注',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '0=待处理 1=通过 2=拒绝',
  `reviewer_uid` varchar(40) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '审批人UID',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_space_uid` (`space_id`,`uid`),
  KEY `idx_space_status` (`space_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Space加入申请记录';

--
-- Table structure for table `space_member`
--

DROP TABLE IF EXISTS `space_member`;
CREATE TABLE `space_member` (
  `id` int NOT NULL AUTO_INCREMENT,
  `space_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `role` smallint NOT NULL DEFAULT '0',
  `status` smallint NOT NULL DEFAULT '1',
  `version` bigint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `spacemember_spaceid_uid` (`space_id`,`uid`),
  KEY `spacemember_uid` (`uid`),
  KEY `spacemember_spaceid_status` (`space_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `thread`
--

DROP TABLE IF EXISTS `thread`;
CREATE TABLE `thread` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `short_id` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '子区独立ID (snowflake)',
  `group_no` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '父群编号',
  `name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '子区名称',
  `creator_uid` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '创建者UID',
  `source_message_id` bigint DEFAULT NULL COMMENT '来源消息ID (可选)',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态: 1=活跃, 2=已归档, 3=已删除',
  `version` bigint NOT NULL DEFAULT '0' COMMENT '版本号',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `message_count` bigint NOT NULL DEFAULT '0' COMMENT '消息数量',
  `last_message_at` timestamp NULL DEFAULT NULL COMMENT '最后一条消息时间',
  `last_message_content` varchar(500) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '最后一条消息内容',
  `last_message_sender_uid` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '最后一条消息发送者UID',
  `thread_md` text COLLATE utf8mb4_unicode_ci COMMENT '子区 GROUP.md 内容',
  `thread_md_version` bigint NOT NULL DEFAULT '0' COMMENT '子区 GROUP.md 版本号（每次更新自增）',
  `thread_md_updated_at` timestamp NULL DEFAULT NULL COMMENT '子区 GROUP.md 最后更新时间',
  `thread_md_updated_by` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '子区 GROUP.md 最后更新者 UID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_short_id` (`short_id`),
  UNIQUE KEY `uk_group_short` (`group_no`,`short_id`),
  KEY `idx_group_no` (`group_no`),
  KEY `idx_creator` (`creator_uid`),
  KEY `idx_status` (`status`),
  KEY `idx_status_last_msg_id` (`status`,`last_message_at`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='群组子区表';

--
-- Table structure for table `thread_member`
--

DROP TABLE IF EXISTS `thread_member`;
CREATE TABLE `thread_member` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `thread_id` bigint unsigned NOT NULL COMMENT '子区ID',
  `uid` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '用户UID',
  `role` tinyint NOT NULL DEFAULT '0' COMMENT '角色: 0=普通成员, 1=创建者',
  `version` bigint NOT NULL DEFAULT '0' COMMENT '版本号',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_thread_uid` (`thread_id`,`uid`),
  KEY `idx_uid` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='子区成员表';

--
-- Table structure for table `thread_setting`
--

DROP TABLE IF EXISTS `thread_setting`;
CREATE TABLE `thread_setting` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `group_no` varchar(40) NOT NULL DEFAULT '' COMMENT '父群编号',
  `short_id` varchar(32) NOT NULL DEFAULT '' COMMENT '子区 shortID',
  `uid` varchar(40) NOT NULL DEFAULT '' COMMENT '用户 UID',
  `mute` tinyint NOT NULL DEFAULT '0' COMMENT '免打扰: 0=关闭, 1=开启',
  `version` bigint NOT NULL DEFAULT '0' COMMENT '版本号',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_thread_uid` (`group_no`,`short_id`,`uid`),
  KEY `idx_uid` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='子区用户设置表';

--
-- Table structure for table `user`
--

DROP TABLE IF EXISTS `user`;
CREATE TABLE `user` (
  `id` int NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `short_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `short_status` smallint NOT NULL DEFAULT '0',
  `sex` smallint NOT NULL DEFAULT '0',
  `robot` smallint NOT NULL DEFAULT '0',
  `category` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `role` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `username` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `password` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `zone` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `phone` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `chat_pwd` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `lock_screen_pwd` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `lock_after_minute` int NOT NULL DEFAULT '0',
  `vercode` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `is_upload_avatar` smallint NOT NULL DEFAULT '0',
  `qr_vercode` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `device_lock` smallint NOT NULL DEFAULT '0',
  `search_by_phone` smallint NOT NULL DEFAULT '1',
  `search_by_short` smallint NOT NULL DEFAULT '1',
  `new_msg_notice` smallint NOT NULL DEFAULT '1',
  `msg_show_detail` smallint NOT NULL DEFAULT '1',
  `voice_on` smallint NOT NULL DEFAULT '1',
  `shock_on` smallint NOT NULL DEFAULT '1',
  `mute_of_app` smallint NOT NULL DEFAULT '0',
  `offline_protection` smallint NOT NULL DEFAULT '0',
  `version` bigint NOT NULL DEFAULT '0',
  `status` smallint NOT NULL DEFAULT '1',
  `bench_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `app_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'app id',
  `email` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'email地址',
  `is_destroy` smallint NOT NULL DEFAULT '0' COMMENT '是否已销毁',
  `wx_openid` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '微信openid',
  `wx_unionid` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '微信unionid',
  `gitee_uid` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'gitee的用户id',
  `github_uid` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'github的用户id',
  `web3_public_key` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'web3公钥',
  `msg_expire_second` bigint NOT NULL DEFAULT '0' COMMENT '消息过期时长(单位秒)',
  `destroy_apply_at` datetime DEFAULT NULL COMMENT '注销申请时间',
  `destroy_expire_at` datetime DEFAULT NULL COMMENT '注销到期执行时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid` (`uid`),
  UNIQUE KEY `short_no_udx` (`short_no`),
  KEY `idx_user_destroy_expire` (`is_destroy`,`destroy_expire_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `user_api_key`
--

DROP TABLE IF EXISTS `user_api_key`;
CREATE TABLE `user_api_key` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) NOT NULL DEFAULT '' COMMENT '用户UID',
  `api_key` varchar(100) NOT NULL DEFAULT '' COMMENT 'API Key (uk_ prefix)',
  `space_id` varchar(40) NOT NULL DEFAULT '' COMMENT '绑定的Space ID',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_api_key` (`api_key`),
  KEY `idx_uid` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户API Key';

--
-- Table structure for table `user_last_offset`
--

DROP TABLE IF EXISTS `user_last_offset`;
CREATE TABLE `user_last_offset` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `channel_type` smallint NOT NULL DEFAULT '0',
  `message_seq` bigint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid_user_last_offset_unidx` (`uid`,`channel_id`,`channel_type`),
  KEY `uid_user_last_offset_idx` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `user_maillist`
--

DROP TABLE IF EXISTS `user_maillist`;
CREATE TABLE `user_maillist` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `phone` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `zone` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `name` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `vercode` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid_maillist_index` (`uid`,`zone`,`phone`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `user_oidc_identity`
--

DROP TABLE IF EXISTS `user_oidc_identity`;
CREATE TABLE `user_oidc_identity` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) NOT NULL DEFAULT '',
  `issuer` varchar(255) NOT NULL DEFAULT '',
  `subject` varchar(255) NOT NULL DEFAULT '',
  `email` varchar(255) NOT NULL DEFAULT '',
  `email_verified` smallint NOT NULL DEFAULT '0',
  `phone` varchar(32) NOT NULL DEFAULT '',
  `phone_verified` smallint NOT NULL DEFAULT '0',
  `linked_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_login_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_issuer_subject` (`issuer`,`subject`),
  KEY `idx_uid` (`uid`),
  KEY `idx_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

--
-- Table structure for table `user_oidc_refresh`
--

DROP TABLE IF EXISTS `user_oidc_refresh`;
CREATE TABLE `user_oidc_refresh` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `identity_id` bigint NOT NULL,
  `token_hash` char(64) NOT NULL,
  `token_ciphertext` varbinary(4096) NOT NULL,
  `expires_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `last_refreshed_at` timestamp NULL DEFAULT NULL,
  `revoked_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_token_hash` (`token_hash`),
  KEY `idx_identity` (`identity_id`),
  KEY `idx_expires` (`expires_at`,`revoked_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

--
-- Table structure for table `user_online`
--

DROP TABLE IF EXISTS `user_online`;
CREATE TABLE `user_online` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `device_flag` smallint NOT NULL DEFAULT '0',
  `last_online` int NOT NULL DEFAULT '0',
  `last_offline` int NOT NULL DEFAULT '0',
  `online` tinyint(1) NOT NULL DEFAULT '0',
  `version` bigint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid_device` (`uid`,`device_flag`),
  KEY `online_idx` (`online`),
  KEY `uid_idx` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `user_pinned_channel`
--

DROP TABLE IF EXISTS `user_pinned_channel`;
CREATE TABLE `user_pinned_channel` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) NOT NULL COMMENT '用户ID',
  `space_id` varchar(40) NOT NULL DEFAULT '' COMMENT '空间ID，空字符串表示全局',
  `channel_id` varchar(100) NOT NULL COMMENT '频道ID',
  `channel_type` tinyint NOT NULL COMMENT '频道类型: 1私聊 2群 5子区',
  `sort_order` int DEFAULT '0' COMMENT '排序值',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_space_channel` (`uid`,`space_id`,`channel_id`,`channel_type`),
  KEY `idx_uid_space_sort` (`uid`,`space_id`,`sort_order`),
  KEY `idx_channel` (`channel_id`,`channel_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户置顶频道（Space隔离）';

--
-- Table structure for table `user_red_dot`
--

DROP TABLE IF EXISTS `user_red_dot`;
CREATE TABLE `user_red_dot` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `count` smallint NOT NULL DEFAULT '0',
  `category` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `is_dot` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_red_dot_uid_categoryx` (`uid`,`category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `user_setting`
--

DROP TABLE IF EXISTS `user_setting`;
CREATE TABLE `user_setting` (
  `id` int NOT NULL AUTO_INCREMENT,
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `to_uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `mute` smallint NOT NULL DEFAULT '0',
  `top` smallint NOT NULL DEFAULT '0',
  `blacklist` smallint NOT NULL DEFAULT '0',
  `chat_pwd_on` smallint NOT NULL DEFAULT '0',
  `screenshot` smallint NOT NULL DEFAULT '1',
  `revoke_remind` smallint NOT NULL DEFAULT '1',
  `receipt` smallint NOT NULL DEFAULT '1',
  `version` bigint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `remark` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户备注',
  `flame` smallint NOT NULL DEFAULT '0' COMMENT '阅后即焚是否开启 1.开启 0.未开启',
  `flame_second` smallint NOT NULL DEFAULT '0' COMMENT '阅后即焚销毁秒数',
  PRIMARY KEY (`id`),
  UNIQUE KEY `to_uid_uid` (`uid`,`to_uid`),
  KEY `uid_idx` (`uid`),
  KEY `idx_user_setting_to_uid` (`to_uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `user_verification`
--

DROP TABLE IF EXISTS `user_verification`;
CREATE TABLE `user_verification` (
  `user_id` varchar(40) NOT NULL COMMENT 'OCTO 用户 UID',
  `real_name` varchar(128) NOT NULL COMMENT '实名（CAS/企微/飞书 返回）',
  `source` varchar(32) NOT NULL COMMENT '实名来源: cas/wecom/feishu',
  `source_sub` varchar(128) NOT NULL COMMENT '来源侧 sub（如 CAS user_id）',
  `emp_id` varchar(64) DEFAULT NULL COMMENT '工号（可空）',
  `dept` varchar(255) DEFAULT NULL COMMENT '部门（可空）',
  `email` varchar(255) DEFAULT NULL COMMENT '邮箱（可空）',
  `mobile` varchar(32) DEFAULT NULL COMMENT '手机号（可空）',
  `verified_at` datetime NOT NULL COMMENT '实名完成时间（UTC）',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '记录更新时间',
  PRIMARY KEY (`user_id`),
  KEY `idx_user_verification_source` (`source`,`source_sub`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户实名认证（OCTO 实名链路）';

--
-- Table structure for table `user_voice_context`
--

DROP TABLE IF EXISTS `user_voice_context`;
CREATE TABLE `user_voice_context` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `uid` varchar(100) NOT NULL COMMENT 'bot owner uid',
  `space_id` varchar(100) NOT NULL COMMENT 'Space ID',
  `asr_correct_context` text NOT NULL COMMENT '纠错上下文内容（最大10000字符）',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `updated_by` varchar(100) NOT NULL COMMENT '设置该上下文的 bot id 或 user uid',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_uid_space` (`uid`,`space_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户语音纠错上下文';

--
-- Table structure for table `workplace_app`
--

DROP TABLE IF EXISTS `workplace_app`;
CREATE TABLE `workplace_app` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `app_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `icon` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `description` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `app_category` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `status` smallint NOT NULL DEFAULT '1',
  `jump_type` smallint NOT NULL DEFAULT '0',
  `app_route` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `web_route` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `is_paid_app` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `workplace_app_appid` (`app_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `workplace_app_user_record`
--

DROP TABLE IF EXISTS `workplace_app_user_record`;
CREATE TABLE `workplace_app_user_record` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `app_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `count` int NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `workplace_app_user_record_uid_appid` (`uid`,`app_id`),
  KEY `workplace_app_user_record_uid` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `workplace_banner`
--

DROP TABLE IF EXISTS `workplace_banner`;
CREATE TABLE `workplace_banner` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `banner_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `cover` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `title` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `description` varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `jump_type` smallint NOT NULL DEFAULT '0',
  `route` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `sort_num` int NOT NULL DEFAULT '0' COMMENT '排序号',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `workplace_category`
--

DROP TABLE IF EXISTS `workplace_category`;
CREATE TABLE `workplace_category` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `category_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `sort_num` int NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `workplace_category_app`
--

DROP TABLE IF EXISTS `workplace_category_app`;
CREATE TABLE `workplace_category_app` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `category_no` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `app_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `sort_num` int NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `workplace_category_app_cno_aid` (`category_no`,`app_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `workplace_user_app`
--

DROP TABLE IF EXISTS `workplace_user_app`;
CREATE TABLE `workplace_user_app` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `app_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `sort_num` int NOT NULL DEFAULT '0',
  `uid` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `workplace_user_app_uid` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;



-- ============================================================
-- gorp_migrations seed (thread-* filtered out — conditional module)
-- ============================================================

INSERT INTO `gorp_migrations` VALUES ('app-20201103-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('app-20230912-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('app_bot-20260505-01.sql','2026-05-09 11:56:41');
INSERT INTO `gorp_migrations` VALUES ('app_bot-20260508-01.sql','2026-05-09 11:56:41');
INSERT INTO `gorp_migrations` VALUES ('app_bot-20260509-01.sql','2026-05-09 12:39:27');
INSERT INTO `gorp_migrations` VALUES ('app_bot-20260510-01.sql','2026-05-10 17:55:01');
INSERT INTO `gorp_migrations` VALUES ('app_bot-20260510-02.sql','2026-05-10 18:29:26');
INSERT INTO `gorp_migrations` VALUES ('backup-20260331-01.sql','2026-03-31 18:40:48');
INSERT INTO `gorp_migrations` VALUES ('backup-20260401-01.sql','2026-04-01 14:28:55');
INSERT INTO `gorp_migrations` VALUES ('botfather-20260226-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('botfather-20260318-01.sql','2026-03-18 16:42:51');
INSERT INTO `gorp_migrations` VALUES ('botfather-20260318-02.sql','2026-03-18 18:00:22');
INSERT INTO `gorp_migrations` VALUES ('botfather-20260324-01.sql','2026-03-25 08:54:37');
INSERT INTO `gorp_migrations` VALUES ('botfather-20260326-01.sql','2026-03-26 20:35:39');
INSERT INTO `gorp_migrations` VALUES ('botfather-20260417-01.sql','2026-04-18 16:33:54');
INSERT INTO `gorp_migrations` VALUES ('bot_api-20260505.sql','2026-05-09 11:56:41');
INSERT INTO `gorp_migrations` VALUES ('category-20260403-01.sql','2026-04-08 14:22:29');
INSERT INTO `gorp_migrations` VALUES ('category-20260415-01.sql','2026-04-15 17:08:50');
INSERT INTO `gorp_migrations` VALUES ('category-20260416-01.sql','2026-04-16 16:19:09');
INSERT INTO `gorp_migrations` VALUES ('category-20260418-01.sql','2026-04-18 16:33:54');
INSERT INTO `gorp_migrations` VALUES ('category-20260428-01.sql','2026-04-29 01:57:01');
INSERT INTO `gorp_migrations` VALUES ('channel-20221124-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('channel-20230920-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('channel-20240515-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('common-20210421-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('common-20210818-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('common-20211108-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('common-20220908-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('common-20220916-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('common-20220917-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('common-20221111-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('common-20221114-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('common-20230203-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('common-20240418-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('common-20240506-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('common-20240510-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('common-20240528-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('common-20260408-01.sql','2026-04-08 18:23:27');
INSERT INTO `gorp_migrations` VALUES ('common-20260427-01.sql','2026-04-27 14:21:16');
INSERT INTO `gorp_migrations` VALUES ('event-20191106-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('event-20250423-01.sql','2026-03-13 18:20:02');
INSERT INTO `gorp_migrations` VALUES ('group_20191106-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('group_20211202-02.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('group_20220411-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('group_20220815-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('group_20220818-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('group_20220830-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('group_20231123-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('group_20240510-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('group_20260318-01.sql','2026-03-20 22:10:20');
INSERT INTO `gorp_migrations` VALUES ('group_20260424-01.sql','2026-04-25 09:42:12');
INSERT INTO `gorp_migrations` VALUES ('group_20260425-01.sql','2026-04-25 13:19:20');
INSERT INTO `gorp_migrations` VALUES ('message-20210305-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('message-20210407-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('message-20210416-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('message-20210813-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('message-20211027-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('message-20220414-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('message-20220418-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('message-20220422-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('message-20220801-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('message-20220810-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('message-20221122-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('message-20240510-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('message-20250624-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('oidc-20260427-01.sql','2026-04-27 15:11:19');
INSERT INTO `gorp_migrations` VALUES ('oidc-20260428-01.sql','2026-04-28 11:45:21');
INSERT INTO `gorp_migrations` VALUES ('report-202012221237-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('report-202211291659-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('robot-20210926-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('robot-20211026-01.sql','2026-03-13 18:20:03');
INSERT INTO `gorp_migrations` VALUES ('robot-20211105-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('robot-20260226-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('robot-20260307-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('robot-20260308-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('robot-20260309-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('space-20260307-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('space-20260307-02.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('space-20260307-03.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('space-20260308-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('space-20260308-02.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('space-20260308-03.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('space-20260310-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('space-20260310-02.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('space-20260410-01.sql','2026-04-10 18:51:04');
INSERT INTO `gorp_migrations` VALUES ('space-20260410-02.sql','2026-04-10 18:51:04');
INSERT INTO `gorp_migrations` VALUES ('space-20260423-01.sql','2026-04-23 16:36:23');
INSERT INTO `gorp_migrations` VALUES ('space-20260424-01.sql','2026-04-27 13:06:25');
INSERT INTO `gorp_migrations` VALUES ('user-20191106-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20210204-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20210405-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20210413-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20210907-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20210916-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20211115-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20220222-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20220609-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20220713-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20220816-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20220906-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20220919-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20230611-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20230911-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20230924-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20231127-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20260228-01.sql','2026-03-13 18:20:04');
INSERT INTO `gorp_migrations` VALUES ('user-20260305-01.sql','2026-03-13 18:20:05');
INSERT INTO `gorp_migrations` VALUES ('user-20260424-01.sql','2026-04-24 18:16:36');
INSERT INTO `gorp_migrations` VALUES ('user-20260427-01.sql','2026-04-27 14:21:16');
INSERT INTO `gorp_migrations` VALUES ('user-20260505-01.sql','2026-05-06 00:24:02');
INSERT INTO `gorp_migrations` VALUES ('user-20260510-01.sql','2026-05-10 22:38:02');
INSERT INTO `gorp_migrations` VALUES ('voice-20260409-01.sql','2026-04-10 14:08:59');
INSERT INTO `gorp_migrations` VALUES ('webhook-20210226-01.sql','2026-03-13 18:20:05');
INSERT INTO `gorp_migrations` VALUES ('webhook-20230920-01.sql','2026-03-13 18:20:05');
INSERT INTO `gorp_migrations` VALUES ('webhook-20241217-01.sql','2026-03-13 18:20:05');
INSERT INTO `gorp_migrations` VALUES ('workplace-20230823-01.sql','2026-03-13 18:20:05');
INSERT INTO `gorp_migrations` VALUES ('workplace-20230906-01.sql','2026-03-13 18:20:05');
INSERT INTO `gorp_migrations` VALUES ('workplace-20240113-01.sql','2026-03-13 18:20:05');

-- 118 migrations seeded, 6 thread-* skipped: thread-20260402-01.sql, thread-20260402-02.sql, thread-20260410-01.sql, thread-20260413-01.sql, thread-20260422-01.sql, thread-20260511-01.sql

SET FOREIGN_KEY_CHECKS = 1;
SET UNIQUE_CHECKS = 1;

SELECT 'octo-db-ready' AS status,
       (SELECT COUNT(*) FROM information_schema.tables
        WHERE table_schema = DATABASE()) AS tables,
       (SELECT COUNT(*) FROM gorp_migrations) AS seeded_migrations;
