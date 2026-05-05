# OCTO 实名认证链路（YUJ-354 / GH#1300）

> 闭合 OCTO 后端 ↔ dmwork-verify-service（accounts.example.com）的实名回写链路。

## 总体链路

```
OCTO 前端                OCTO 后端                       verify-service (CAS)
   │                        │                                  │
   │  POST /internal/       │                                  │
   │  verify-token          │                                  │
   │───────────────────────▶│                                  │
   │  {token, verify_url}   │                                  │
   │◀───────────────────────│                                  │
   │                        │                                  │
   │       用户点跳转 verify_url (带 5min HS256 JWT)            │
   │───────────────────────────────────────────────────────────▶│
   │                                                            │ 走 CAS 实名
   │                        │                                  │
   │                        │ POST /v1/internal/verification/  │
   │                        │ complete (HMAC-SHA256 签名)       │
   │                        │◀─────────────────────────────────│
   │                        │ 200 {ok:true}                    │
   │                        │─────────────────────────────────▶│
   │                        │                                  │
   │   GET /v1/users/:uid   │                                  │
   │───────────────────────▶│                                  │
   │  {realname_verified,   │                                  │
   │   real_name, ...}      │                                  │
   │◀───────────────────────│                                  │
```

## 接口

### 1. POST `/v1/internal/verification/complete`

由 verify-service 调用。**无 OCTO session，走 HMAC 签名鉴权。**

Header：

| Header | 示例 | 说明 |
| --- | --- | --- |
| `Content-Type` | `application/json` | |
| `X-OCTO-Signature` | `sha256=<hex>` | `hmac_sha256(body, OCTO_INTERNAL_HMAC_SECRET)` |

Body：

```json
{
  "octo_user_id": "u_abc123",
  "real_name":    "张三",
  "cas_user_id":  "31",
  "emp_id":       null,
  "dept":         null,
  "email":        "zhangsan@example.com",
  "mobile":       "13000000000",
  "verified_at":  "2026-05-05T14:55:49Z",
  "source":       "cas"
}
```

| 字段 | 类型 | 必填 | 备注 |
| --- | --- | --- | --- |
| octo_user_id | string | Y | OCTO `user.uid` |
| real_name | string | Y | CAS 真名 |
| cas_user_id | string | Y* | source 侧 sub；for source=cas 必填 |
| emp_id | string\|null | N | |
| dept | string\|null | N | |
| email | string\|null | N | |
| mobile | string\|null | N | |
| verified_at | string (RFC3339 UTC) | Y | |
| source | `cas` \| `wecom` \| `feishu` | Y | |

响应码：

| Code | 含义 |
| --- | --- |
| 200 | `{"ok": true}` |
| 400 | body 畸形 / 必填缺失 / verified_at 格式错 |
| 401 | HMAC 不匹配 或 服务端 `OCTO_INTERNAL_HMAC_SECRET` 未配 |
| 404 | `octo_user_id` 在 OCTO user 表不存在 |
| 500 | DB / 其他异常 |

幂等性：表以 `user_id` 为主键，重复回调按最新值覆盖。

### 2. POST `/v1/internal/verify-token`

由 OCTO 前端调用。**需 OCTO session**（走 AuthMiddleware），签发 5 分钟短时 JWT。

Body（可选）：

```json
{ "return_to": "https://api.example.com/home" }
```

响应：

```json
{
  "token": "<HS256 JWT>",
  "verify_url": "https://accounts.example.com/verify?token=<JWT>&return_to=<return_to>",
  "expires_at": 1735862349
}
```

JWT claims：

```json
{
  "sub":     "<octo_user_id>",
  "purpose": "verify",
  "iat":     <now>,
  "exp":     <now+300>
}
```

算法：HS256。密钥：`OCTO_JWT_SECRET`（与 verify-service 共享）。

## 数据库

表 `user_verification`（migration `user-20260505-01.sql`）：

| 列 | 类型 | 说明 |
| --- | --- | --- |
| user_id | VARCHAR(40) PK | OCTO 用户 UID |
| real_name | VARCHAR(128) NOT NULL | |
| source | VARCHAR(32) NOT NULL | cas/wecom/feishu |
| source_sub | VARCHAR(128) NOT NULL | 来源 sub |
| emp_id | VARCHAR(64) NULL | |
| dept | VARCHAR(255) NULL | |
| email | VARCHAR(255) NULL | |
| mobile | VARCHAR(32) NULL | |
| verified_at | DATETIME NOT NULL | UTC |
| updated_at | DATETIME ON UPDATE CURRENT_TIMESTAMP | |

## Profile API 增量

现有 `GET /v1/users/:uid` / `POST /v1/users` 返回的 `UserDetailResp` 新增字段：

| 字段 | 类型 | 含义 |
| --- | --- | --- |
| `realname_verified` | bool | 是否已实名 |
| `real_name` | string（已认证才有） | 实名姓名 |

未实名用户 `realname_verified=false`、`real_name` 省略（`omitempty`）。

## 环境变量

| Env | 必填 | 默认 | 说明 |
| --- | --- | --- | --- |
| `OCTO_INTERNAL_HMAC_SECRET` | Y（启用本链路时） | — | 与 verify-service 共享的 HMAC 密钥。未配时 `/v1/internal/verification/complete` 恒回 401（fail-closed）。 |
| `OCTO_JWT_SECRET` | Y（启用本链路时） | — | 与 verify-service 共享的 HS256 JWT 密钥。未配时 `/v1/internal/verify-token` 恒回 503。 |
| `OCTO_VERIFY_URL_BASE` | N | `https://accounts.example.com/verify` | verify-service 跳转基址。 |
| `OCTO_VERIFY_RETURN_TO_DEFAULT` | N | 空 | 客户端未传 `return_to` 时的默认值；为空则不挂参数。 |

运维操作：从 verify-service 的 `.env` 拷贝 `OCTO_INTERNAL_HMAC_SECRET` 与 `OCTO_JWT_SECRET` 到 `dmworkim` 启动环境（k8s Secret / systemd EnvironmentFile / docker-compose `.env`）。

## 验证命令

```bash
# 1) 生成 HMAC 签名
BODY='{"octo_user_id":"u_abc","real_name":"张三","cas_user_id":"31","verified_at":"2026-05-05T14:55:49Z","source":"cas"}'
SIG="sha256=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$OCTO_INTERNAL_HMAC_SECRET" -hex | awk '{print $NF}')"

# 2) 成功 case
curl -i -X POST https://api.example.com/v1/internal/verification/complete \
  -H "Content-Type: application/json" \
  -H "X-OCTO-Signature: $SIG" \
  -d "$BODY"
# 期望: 200 {"ok":true}

# 3) 错误 HMAC
curl -i -X POST https://api.example.com/v1/internal/verification/complete \
  -H "Content-Type: application/json" \
  -H "X-OCTO-Signature: sha256=deadbeef" \
  -d "$BODY"
# 期望: 401

# 4) OCTO 前端签发 token（需登录）
curl -X POST https://api.example.com/v1/internal/verify-token \
  -H "Authorization: <OCTO session token>" \
  -H "Content-Type: application/json" \
  -d '{"return_to":"https://api.example.com/me"}'
# 期望: {"token":"...","verify_url":"https://accounts.example.com/verify?token=...&return_to=...","expires_at":...}
```

## 安全要点

- **fail-closed**：HMAC/JWT secret 任一未配置时，相关接口不开放（401 / 503）。
- **HMAC 比较**走常时 `hmac.Equal`，防时序侧信道。
- `/internal/*` 路径应在网关（nginx / ingress）限定为内网可达作为防御纵深。
- JWT 目前 HS256 + 共享 secret（MVP）。后续切换到 RS256 仅需：verify-service 发布 JWKS，OCTO 改用 RSA 私钥签、公钥由 verify-service 在 /.well-known/jwks.json 验。接口契约不变。
- `purpose=verify` 固定 claim 防止 token 被误用到其他鉴权面。

## 相关

- verify-service repo：`Mininglamp-OSS/octo-verify-service`（内部）
- GH Issue：Mininglamp-OSS/octo-server#1300
- Multica Issue：YUJ-354
