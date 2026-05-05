package user

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Mininglamp-OSS/octo-lib/pkg/wkhttp"
	"github.com/gocraft/dbr/v2"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

// =============================================================================
// OCTO 实名认证链路接口（YUJ-354 / GH#1300）
//
// 两个接口：
//   1. POST /v1/internal/verification/complete
//      由 dmwork-verify-service（accounts.example.com）在用户实名完成后回调，
//      以 HMAC-SHA256 签名体认证，upsert 到 user_verification。
//
//   2. POST /v1/internal/verify-token
//      由 OCTO 前端调用，基于当前登录会话签发 5 分钟短时 HS256 JWT，
//      返回 token 与跳转 verify-service 的 URL。
//
// 关键安全约束：
//   - 两个密钥均与 verify-service 共享，来自环境变量：
//       OCTO_INTERNAL_HMAC_SECRET  HMAC 体签名（对称）
//       OCTO_JWT_SECRET            HS256 JWT 签名（对称）
//   - 未配置时 fail-closed：直接 503，不降级到无鉴权。
//   - HMAC 和 hex 比较均使用 hmac.Equal / subtle 常时比较，防时序侧信道。
// =============================================================================

const (
	// X-OCTO-Signature: sha256=<hex> —— 由 verify-service 发送，用 OCTO_INTERNAL_HMAC_SECRET 签 body。
	octoSignatureHeader = "X-OCTO-Signature"
	// JWT purpose claim 固定值，防 token 被跨用途重放到其他服务。
	octoVerifyJWTPurpose = "verify"
	// 5 分钟短时 token —— 用户点"去实名" → 跳 verify-service 完成 CAS 授权往返的窗口。
	octoVerifyJWTTTL = 5 * time.Minute
	// verify-service 默认跳转地址。env 覆盖：OCTO_VERIFY_URL_BASE。
	octoVerifyURLBaseDefault = "https://accounts.example.com/verify"
	// /v1/internal/verification/complete body 上限 —— verify-service 实际 payload ~500B，
	// 1 MB 非常宽松，但能挡住攻击者用 multi-GB body 在 HMAC 验签前吃满内存的 DoS。
	octoCompleteMaxBody = 1 << 20 // 1 MiB
)

// octoSourceAllowlist 限定 verify-service 上游允许的 source 取值。
// 与 verify-service / user_verification 表契约保持一致（cas / wecom / feishu）。
var octoSourceAllowlist = map[string]struct{}{
	"cas":    {},
	"wecom":  {},
	"feishu": {},
}

// octoReturnToAllowedSchemes 列出 /v1/internal/verify-token 允许回跳的 scheme 前缀。
// 其他 scheme（javascript:/data:/file: 等）一律忽略，让 verify-service 走默认 return_to，
// 避免把 open-redirect / XSS 向量透传到前端。
var octoReturnToAllowedSchemes = []string{
	"https://",
	"octo://",
	"dmwork://",
}

// verificationCompleteReq 与 verify-service 发送的 JSON 体一一对应（详见 GH#1300）。
type verificationCompleteReq struct {
	OCTOUserID string `json:"octo_user_id"`
	RealName   string `json:"real_name"`
	CASUserID  string `json:"cas_user_id"`
	EmpID      string `json:"emp_id"`
	Dept       string `json:"dept"`
	Email      string `json:"email"`
	Mobile     string `json:"mobile"`
	VerifiedAt string `json:"verified_at"` // RFC3339
	Source     string `json:"source"`      // cas/wecom/feishu
}

type verifyTokenResp struct {
	Token     string `json:"token"`
	VerifyURL string `json:"verify_url"`
	ExpiresAt int64  `json:"expires_at"` // unix seconds，方便前端做倒计时
}

// verifyTokenReq OCTO 前端可选地传一个 return_to，用于覆盖默认回跳地址。
// verify-service 会把 return_to 原样挂在 state 上，完成后带回来。
type verifyTokenReq struct {
	ReturnTo string `json:"return_to,omitempty"`
}

// octoVerifyJWTClaims 见 GH#1300 JWT claims 定义。
type octoVerifyJWTClaims struct {
	Purpose string `json:"purpose"`
	jwt.RegisteredClaims
}

// routeVerification 注册 OCTO 实名认证链路两个接口。
//
// 路径遵循项目既定约定 `/v1/internal/*`（与 modules/notify/api.go 对齐）：
//   - /v1/internal/verification/complete 由 verify-service 回调；
//   - /v1/internal/verify-token 由 OCTO 前端调用。
//
// nginx / 网关仅反代 /v1/* 前缀，因此所有内网路径必须挂在 /v1/internal 下；
// PR#1301 误用 `/internal/*` 会被 nginx 直接返回 405（参见 GH#1302）。
//
// 网络侧隔离：两条路径均应在 nginx / 网关上限定为内网可达（运维 Runbook 说明），
// 应用层的 HMAC / 登录态鉴权是防御纵深第二层。
func (u *User) routeVerification(r *wkhttp.WKHttp) {
	internal := r.Group("/v1/internal")
	{
		// HMAC 认证：由 verify-service → OCTO；不要套 AuthMiddleware，verify-service 没有 OCTO session。
		internal.POST("/verification/complete", u.verificationComplete)
	}
	// verify-token 必须已登录，用 OCTO session 绑 sub；挂在 /v1/internal 下 + AuthMiddleware 即可。
	authInternal := r.Group("/v1/internal", u.ctx.AuthMiddleware(r))
	{
		authInternal.POST("/verify-token", u.issueVerifyToken)
	}
}

// verificationComplete 处理 verify-service 的实名完成回调。
//
// 流程：
//  1. 读 body（同时缓存以便后续 JSON 解析，不改变 HMAC 验签基准）
//  2. 校 X-OCTO-Signature = HMAC-SHA256(body, OCTO_INTERNAL_HMAC_SECRET)
//  3. JSON 解析 + 必填字段校验
//  4. 确认 OCTO 用户存在（不存在 → 404，让 verify-service 记 audit log 人工追查）
//  5. Upsert user_verification
//
// 错误码：
//
//	400 body 畸形
//	401 HMAC 不匹配 / secret 未配
//	404 OCTO 用户不存在
//	500 其他异常
func (u *User) verificationComplete(c *wkhttp.Context) {
	secret := strings.TrimSpace(os.Getenv("OCTO_INTERNAL_HMAC_SECRET"))
	if secret == "" {
		// fail-closed：secret 未配时直接拒绝，避免"配置漏掉但接口开放"的安全坑。
		u.Warn("OCTO_INTERNAL_HMAC_SECRET 未配置，拒绝 /v1/internal/verification/complete 请求")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin401Body("internal auth not configured"))
		return
	}

	// 先用 MaxBytesReader 硬封顶，既能在 HMAC 验签前阻断 multi-GB body DoS，
	// 也让底层把连接的读超时/长度超限传给 gin 正确返回 413/400。
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, octoCompleteMaxBody)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		// io.ReadAll 在超过 MaxBytesReader 时返回 "http: request body too large"。
		u.Warn("读取请求体失败（或超过上限）",
			zap.Error(err),
			zap.String("path", c.Request.URL.Path),
			zap.String("remote", c.ClientIP()),
		)
		if strings.Contains(err.Error(), "request body too large") {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin400Body("body too large"))
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin400Body("invalid body"))
		return
	}
	// 允许下游中间件 / 日志再次读取（本处程序逻辑不依赖，保险起见还原）
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	sig := c.GetHeader(octoSignatureHeader)
	if !verifyOCTOSignature(sig, body, secret) {
		u.Warn("HMAC 签名校验失败",
			zap.String("path", c.Request.URL.Path),
			zap.String("remote", c.ClientIP()),
		)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin401Body("bad signature"))
		return
	}

	var req verificationCompleteReq
	if err := json.Unmarshal(body, &req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin400Body("invalid json"))
		return
	}
	if req.OCTOUserID == "" || req.RealName == "" || req.Source == "" || req.VerifiedAt == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin400Body("missing required field"))
		return
	}
	// cas_user_id 用于回查 CAS 账户映射，缺失会让下游运营无法把 OCTO 用户跟员工 ID 对上；
	// verify-service 约定必传，这里补一道 defense-in-depth。
	if req.CASUserID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin400Body("missing cas_user_id"))
		return
	}
	// source allowlist —— 防 verify-service 侧 bug / 误配把未知上游写入 user_verification。
	if _, ok := octoSourceAllowlist[req.Source]; !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin400Body("invalid source"))
		return
	}

	// 校验 OCTO 用户存在 —— 不存在意味着 verify-service 签发的 JWT sub 指向了一个
	// OCTO 侧已注销 / 未迁移的账户，必须上报而不是静默 upsert 一个孤儿记录。
	existing, err := u.db.QueryByUID(req.OCTOUserID)
	if err != nil {
		u.Error("查询 OCTO 用户失败", zap.Error(err), zap.String("uid", req.OCTOUserID))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin500Body("db error"))
		return
	}
	if existing == nil {
		u.Warn("verify-service 回调了不存在的 OCTO 用户",
			zap.String("uid", req.OCTOUserID),
			zap.String("source", req.Source),
		)
		c.AbortWithStatusJSON(http.StatusNotFound, map[string]interface{}{
			"ok":    false,
			"error": "octo_user_id not found",
		})
		return
	}

	verifiedAt, err := time.Parse(time.RFC3339, req.VerifiedAt)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin400Body("invalid verified_at"))
		return
	}

	m := &verificationModel{
		UserID:     req.OCTOUserID,
		RealName:   req.RealName,
		Source:     req.Source,
		SourceSub:  req.CASUserID,
		EmpID:      nullableString(req.EmpID),
		Dept:       nullableString(req.Dept),
		Email:      nullableString(req.Email),
		Mobile:     nullableString(req.Mobile),
		VerifiedAt: verifiedAt.UTC(),
	}
	if err := u.verificationDB.Upsert(m); err != nil {
		u.Error("写入 user_verification 失败", zap.Error(err), zap.String("uid", req.OCTOUserID))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin500Body("db error"))
		return
	}

	u.Info("OCTO 实名认证完成",
		zap.String("uid", req.OCTOUserID),
		zap.String("source", req.Source),
	)
	c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// issueVerifyToken 为当前登录用户签发 5 分钟短时 JWT（HS256），
// 返回前端直接打开即可跳转 verify-service 的 URL。
//
// verify-service 侧约定：
//   - 收到 token → 校 HS256 + OCTO_JWT_SECRET，校 purpose=verify，校 exp，取 sub=octo_user_id。
//   - 完成后走 /v1/internal/verification/complete 回调 OCTO（已由接口 1 处理）。
func (u *User) issueVerifyToken(c *wkhttp.Context) {
	secret := strings.TrimSpace(os.Getenv("OCTO_JWT_SECRET"))
	if secret == "" {
		u.Warn("OCTO_JWT_SECRET 未配置，拒绝签发 verify-token")
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin500Body("verify service not configured"))
		return
	}

	uid := c.GetLoginUID()
	if uid == "" {
		// 走到这里意味着 AuthMiddleware 失效，理论上不该发生；fail-closed。
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin401Body("unauthenticated"))
		return
	}

	// return_to 可选：传则覆盖默认。verify-service 会在 state 中透传，完成后 302 回 return_to。
	var req verifyTokenReq
	if c.Request.ContentLength > 0 {
		// 体可为空；忽略解析错误，让 return_to 回落到默认
		_ = c.BindJSON(&req)
	}

	now := time.Now()
	exp := now.Add(octoVerifyJWTTTL)
	claims := octoVerifyJWTClaims{
		Purpose: octoVerifyJWTPurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uid,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		u.Error("签发 verify JWT 失败", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin500Body("sign error"))
		return
	}

	verifyURL := buildVerifyURL(signed, req.ReturnTo)
	c.Response(verifyTokenResp{
		Token:     signed,
		VerifyURL: verifyURL,
		ExpiresAt: exp.Unix(),
	})
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// verifyOCTOSignature 比对 X-OCTO-Signature。格式：sha256=<hex>
// 前缀不对、解码失败、长度不对或签名不匹配均视为失败；所有比较走常时算法。
func verifyOCTOSignature(header string, body []byte, secret string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	want, err := hex.DecodeString(header[len(prefix):])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	got := mac.Sum(nil)
	return hmac.Equal(got, want)
}

// buildVerifyURL 组装 verify-service 跳转地址。
//
//	base?token=<JWT>[&return_to=<r>]
//
// 不用 net/url.Values.Encode 是为了保持 token 原生 URL-safe 字符不被再编码，
// 与 verify-service 侧 `url.searchParams.get('token')` 解析一致。
func buildVerifyURL(token, returnTo string) string {
	base := strings.TrimSpace(os.Getenv("OCTO_VERIFY_URL_BASE"))
	if base == "" {
		base = octoVerifyURLBaseDefault
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	u := base + sep + "token=" + token
	if returnTo == "" {
		returnTo = strings.TrimSpace(os.Getenv("OCTO_VERIFY_RETURN_TO_DEFAULT"))
	}
	if returnTo != "" && isAllowedReturnToScheme(returnTo) {
		// 校验通过的 return_to 才挂上；QueryEscape 保证冒号、问号等不会破坏 URL 结构。
		// 非法 scheme（javascript:/data:/file: 等）直接丢弃，让 verify-service 走默认 return_to，
		// 杜绝 open-redirect / XSS 透传。
		u += "&return_to=" + url.QueryEscape(returnTo)
	}
	return u
}

// isAllowedReturnToScheme 返回 true 仅当 s 以 octoReturnToAllowedSchemes 中任一 scheme 前缀开头。
// 比较采用小写 —— 防绕过 (`JavaScript:`、`HTTPS://` 等)。
func isAllowedReturnToScheme(s string) bool {
	lower := strings.ToLower(s)
	for _, p := range octoReturnToAllowedSchemes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

func nullableString(s string) dbr.NullString {
	if strings.TrimSpace(s) == "" {
		return dbr.NullString{}
	}
	return dbr.NullString{NullString: sql.NullString{String: s, Valid: true}}
}

func gin400Body(msg string) map[string]interface{} {
	return map[string]interface{}{"ok": false, "error": msg}
}
func gin401Body(msg string) map[string]interface{} {
	return map[string]interface{}{"ok": false, "error": msg}
}
func gin500Body(msg string) map[string]interface{} {
	return map[string]interface{}{"ok": false, "error": msg}
}
