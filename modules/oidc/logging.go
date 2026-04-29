package oidc

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// subHash 返回 OIDC sub claim 的 SHA-256 短哈希(前 8 个 hex 字符)。
//
// 用途:日志和审计 uid 前缀都用它替代明文截断。明文 sub 可能含 IdP 内部用户 id /
// 邮箱形式标识,落到日志或 audit.uid 都属于过度暴露;短哈希保留了"可关联同一
// 用户多次登录"的运维价值,又不可逆推回 IdP 用户身份。
//
// 8 hex(32 bit)冲突空间 ~4.3 亿,排查单条日志足够;真要审计请走 user_oidc_identity 表。
func subHash(sub string) string {
	if sub == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(sub))
	return hex.EncodeToString(sum[:4])
}

// newTraceID 生成 16 hex(8 字节)随机 trace id,贯穿单次 callback 的所有日志。
//
// 不入 wkhttp middleware:OIDC callback 由 IdP 重定向命中,没有上游 X-Request-Id
// 来源,且本期只有 OIDC 路径需要,在入口本地生成更轻量。后续基础设施层补全
// middleware 后,可改为优先读 c.Request.Header.Get("X-Request-Id")。
//
// 失败兜底:crypto/rand 在标准化 OS 上几乎不会失败;真失败时返回固定占位串
// "0000000000000000",日志关联会丢但不阻塞登录。
func newTraceID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(b[:])
}
