package user

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestVerifySignature_EmptySignature 测试空签名导致的 panic 风险 (Issue #396)
func TestVerifySignature_EmptySignature(t *testing.T) {
	// 创建一个最小化的 User 实例，verifySignature 不需要 ctx
	u := &User{}

	testCases := []struct {
		name      string
		signText  string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "empty signature",
			signText:  "",
			expectErr: true,
			errMsg:    "签名数据长度不足",
		},
		{
			name:      "signature too short - 1 byte",
			signText:  "ab",
			expectErr: true,
			errMsg:    "签名数据长度不足",
		},
		{
			name:      "signature too short - 64 bytes",
			signText:  strings.Repeat("ab", 64), // 64 bytes, need 65
			expectErr: true,
			errMsg:    "签名数据长度不足",
		},
	}

	// 使用有效的压缩公钥 (33 字节)
	validPublicKey := "03af80b90d25145da28c583359beb47b21796b2fe1a23c1511e443e7a64dfdb27d"
	verifyText := "test message"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := u.verifySignature(validPublicKey, verifyText, tc.signText)

			if tc.expectErr {
				assert.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.False(t, result)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVerifySignature_ValidLength 测试有效长度签名不会 panic
func TestVerifySignature_ValidLength(t *testing.T) {
	u := &User{}

	// 65 字节签名 (130 hex chars)
	validLengthSignature := strings.Repeat("ab", 65)
	validPublicKey := "03af80b90d25145da28c583359beb47b21796b2fe1a23c1511e443e7a64dfdb27d"
	verifyText := "test message"

	// 不应该 panic，即使签名内容无效
	result, err := u.verifySignature(validPublicKey, verifyText, validLengthSignature)

	// 签名无效但不应该 panic
	assert.NoError(t, err)
	assert.False(t, result) // 签名内容无效，验证失败
}
