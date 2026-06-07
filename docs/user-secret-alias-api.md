# 用户外部密钥别名表 — API Schema / 契约（YUJ-3538，Secrets 1/3 octo-server）

> 本文是 octo-server 产出的对外契约，供 **octo-web**（Secrets 2/3，前端 UI）与
> **openclaw-channel-octo**（Secrets 3/3，channel 插件 write-secret 工具 +
> use-time resolve）对接。

## 背景与信任边界

用户在 Octo 之外自定义管理 token 密钥：

- 用户**不在 Octo 聊天里直接输 key**；key 存在 octo-server（加密落库）。
- 用户在 Octo 里用**自然语言别名**引用某个 key（语音友好，允许中文/空格）。
- bot 经 channel 插件在 **use-time** 调 `resolve` 拿明文，写到目标位置。

**信任边界**：明文 key 不得出现在 Octo 的消息 / 会话 / knowledge graph / 经 Octo
的 LLM 请求里。因此 **任何接口永不回显明文/密文**，明文只在 `resolve` 成功响应体里
出现一次，直达调用方（channel 插件）。

## 核心概念

| 概念 | 说明 |
| --- | --- |
| `secret_id` | 稳定内部 ID（UUID）。**引用锚点**：改名（`display_name`）不断引用。 |
| `display_name` | 用户原始别名短语，语音友好，允许中文/空格/大小写，可重命名。 |
| `display_name` 唯一性 | 按 **normalize**（去空格 + 折叠大小写 + 简繁归一）在 **per-user** 维度判重；撞了报 409 提示换名。 |
| `kind` | 枚举 `llm` / `external`，**仅用于分类过滤展示**，不参与鉴权/解析。 |
| `masked` | 明文尾 4 位掩码（如 `****1234`），供 list 展示，不泄漏完整 key。 |

## 鉴权方式

| 接口组 | 路由前缀 | 鉴权 | owner 判定 |
| --- | --- | --- | --- |
| CRUD（用户态） | `/v1/manager/secrets` | 标准用户会话（`token` header，AuthMiddleware） | owner = 当前登录用户 UID |
| resolve（插件态） | `/v1/bot/secrets/resolve` | **bf_ bot token**（`Authorization: Bearer bf_...`） | owner = 该 bot 的 `creator_uid`，**只返该 owner 的 key** |

- resolve 复用 **bot 凭证体系**：`bf_` bot token 已绑定 owner（`robot.creator_uid`）。
  服务端用 token 反查活跃 User Bot（`status=1`），认定「合法的、代表本用户的
  OpenClaw channel 插件在取」，并把可解析范围**限定到该 owner 本人**。
- anti-enumeration：bot token 无效 / 非 `bf_` 前缀 / 查不到 owner，统一返 `401`，
  不区分具体原因。

---

## 主密钥落点（加密实现说明）

- 算法：**AES-256-GCM**，密文 = 版本前缀 `enc:v1:` + `nonce(12B) || ciphertext || tag(16B)`，
  落 `varbinary(8240)`。复用 octo-server 现有对称加密做法，不自造轮子。明文上限
  8192B，加密后约 8192 + 前缀 7 + nonce 12 + tag 16 = 8227B，列宽 8240 留余量，
  确保最大尺寸 key 不被严格 MySQL 拒插、也不会在非严格模式下静默截断成脏行。
- 主密钥：环境变量 **`OCTO_USER_API_KEY_SECRET`（32 字节）**，与 `modules/botfather`
  的 user-api-key 加密**同一把主密钥落点**。本模块用独立域分离串
  `octo-user-secret-alias-cipher-v1` 经 `HMAC-SHA256(masterKey, domain)` 派生子密钥，
  与 botfather / oidc 的子密钥在密码学上相互独立。
- 主密钥轮换：当前为单密钥（与 botfather user-api-key 现状一致，无内建多版本轮换）。
  轮换需运维替换 `OCTO_USER_API_KEY_SECRET` 并对存量密文重新加密；密文版本前缀
  `enc:v1:` 为未来引入多版本主密钥预留识别位。
- 主密钥缺失/长度非法：模块不阻断进程启动，但所有写接口 / resolve 返回 `500`
  `err.server.usersecret.resolve_failed`，运维补齐 env 后重启即恢复。

---

## 错误码

所有错误响应保留**真实 HTTP 状态码**（无 legacy 固定 400），body 为统一 i18n 信封：

```json
{ "status": 400, "error": { "code": "err.server.usersecret.xxx", "http_status": 400, "details": {} } }
```

| HTTP | code | 触发场景 |
| --- | --- | --- |
| 400 | `err.server.usersecret.request_invalid` | 入参缺失/格式非法（空 `display_name`/`key`、超长、body 解析失败、update 既无 key 又无 display_name） |
| 401 | `err.server.usersecret.unauthorized` | CRUD 未登录；resolve bot 凭证无效 / 认不出合法 owner |
| 404 | `err.server.usersecret.not_found` | CRUD 目标 `secret_id` 不存在（非本 owner 即视为不存在）；**resolve 解引用零命中** |
| 409 | `err.server.usersecret.duplicate_name` | create/rename 时归一化别名撞已有别名（**换名**提示） |
| 422 | `err.server.usersecret.ambiguous` | **resolve 歧义**：匹配到多个候选，返候选列表让上层消歧（见下） |
| 500 | `err.server.usersecret.resolve_failed` | 解引用失败（密文解密/认证失败等内部异常）；主密钥未就绪 |

---

## CRUD 接口（用户态，`/v1/manager/secrets`）

### 1. 新建 — `POST /v1/manager/secrets`

请求：

```json
{ "display_name": "Claude 密钥", "kind": "llm", "key": "sk-xxxxxxxx" }
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `display_name` | string | 是 | 别名短语，≤128 字符 |
| `kind` | string | 否 | `llm` / `external`，缺省/未知归 `external` |
| `key` | string | 是 | 明文 key，≤8192 字符；**仅本次请求体出现，落库即加密** |

响应 `201 Created`（脱敏视图，无明文/密文）：

```json
{
  "secret_id": "a1b2c3d4-...",
  "display_name": "Claude 密钥",
  "kind": "llm",
  "masked": "****xxxx",
  "created_at": "2026-06-07T17:00:00Z",
  "updated_at": "2026-06-07T17:00:00Z"
}
```

错误：`400`（入参非法）、`409`（撞名换名）。

### 2. 列表 — `GET /v1/manager/secrets[?kind=llm|external]`

响应 `200`：

```json
{ "secrets": [ { "secret_id": "...", "display_name": "...", "kind": "llm", "masked": "****xxxx", "created_at": "...", "updated_at": "...", "last_used_at": "..." } ] }
```

`kind` query 可选，按分类过滤。**永不含明文/密文。**

### 3. 更新 — `PUT /v1/manager/secrets/{secret_id}`

承载「换 key」与「重命名」两种动态修改，二选一或同时：

```json
{ "key": "sk-new-rotated" }          // 换 key：只更新密文，secret_id/display_name 不变
{ "display_name": "新名字" }          // 重命名：secret_id/密文不变，改名不断引用
{ "key": "sk-new", "display_name": "新名字" }  // 同时
```

- `key` 与 `display_name` 至少给一个，否则 `400`。
- 响应 `200`：更新后的脱敏视图（结构同 create 响应）。
- 错误：`400`、`404`（`secret_id` 不存在）、`409`（rename 撞名）。

### 4. 删除 — `DELETE /v1/manager/secrets/{secret_id}`

响应 `200 {"status":200}`。错误：`404`（不存在）。

---

## resolve 接口（插件态，use-time）

### `POST /v1/bot/secrets/resolve`

Header：`Authorization: Bearer bf_<bot_token>`

请求：

```json
{ "query": "克劳德密钥" }
```

- `query` 可为 `secret_id`（精确直查）或 `display_name`（精确 + 拼音/模糊匹配）。

**解析规则**：

1. `query` 命中本 owner 某 `secret_id` → 唯一命中，返明文。
2. 否则按名称匹配：
   - 归一化（去空格/小写/简繁）**精确命中** 优先；精确唯一 → 返明文，精确多条 → 歧义。
   - 无精确命中时用**拼音/模糊命中**（语音场景：「克劳德密钥」「我的米要」等可命中
     同写法/同音变体）；唯一 → 返明文，多条 → 歧义。
3. 零命中 → `404`。

**唯一命中** 响应 `200`：

```json
{ "secret_id": "a1b2c3d4-...", "value": "sk-xxxxxxxx" }
```

> `value` 是明文，仅此一处返回，直达调用方，不进任何日志/审计。

**歧义** 响应 `422`（候选列表走统一 i18n 错误信封的 `error.details.candidates`，**不返明文**）：

```json
{
  "status": 422,
  "error": {
    "code": "err.server.usersecret.ambiguous",
    "message": "名称匹配到多个密钥，请进一步区分。",
    "http_status": 422,
    "details": {
      "candidates": [
        { "secret_id": "...", "display_name": "我的密钥", "kind": "external", "masked": "****1111", "created_at": "...", "updated_at": "..." },
        { "secret_id": "...", "display_name": "我的米要", "kind": "external", "masked": "****2222", "created_at": "...", "updated_at": "..." }
      ]
    }
  }
}
```

上层（channel 插件）据 `error.details.candidates` 让用户/agent 选定后，用具体
`secret_id` 重新 `resolve` 拿明文。

**未命中** 响应 `404` `err.server.usersecret.not_found`。
**解引用失败** 响应 `500` `err.server.usersecret.resolve_failed`。

### resolve 审计（P0）

每次 resolve 落一条审计（`user_secret_resolve_audit`）：谁（caller_kind=`user_bot` +
`caller_id`=robot_id）、何时、解了哪个 owner 的哪个 `secret_id`、命中结果
（`ok` / `not_found` / `ambiguous` / `decrypt_fail` / `unauthorized`）。审计为
best-effort（失败仅记日志，不阻塞返回），且**不记录明文/密文**（`query` 截断存储）。

---

## 给下游两仓的对接要点

### octo-web（Secrets 2/3）

- 走 CRUD 四接口（用户会话鉴权）。
- 列表展示用 `display_name` + `kind` + `masked` + 时间元数据；**前端拿不到也不应展示明文**。
- 「换 key」与「重命名」都走 `PUT`（按需传 `key` / `display_name`）。
- 撞名 `409` → 提示用户换名。

### openclaw-channel-octo（Secrets 3/3）

- write-secret 工具：调 CRUD `POST/PUT`（以用户身份），把用户给的 key 存进来。
- use-time resolve：用 bot 自己的 `bf_` token 调 `POST /v1/bot/secrets/resolve`，
  入参用户口述的别名 `query`。
- 处理三态：`200` 唯一命中拿 `value` 写目标位置；`422` 歧义按 `candidates` 消歧后
  用 `secret_id` 再 resolve；`404` 提示「没有这个 key」。
- 明文 `value` 落到指定位置后即用即弃，**不要回写进 Octo 任何消息/上下文**。
