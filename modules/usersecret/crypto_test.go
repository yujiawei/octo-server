package usersecret

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setTestMasterKey 注入一把固定长度合法主密钥(32 字节 ASCII),返回清理函数。
func setTestMasterKey(t *testing.T) func() {
	t.Helper()
	const key = "0123456789abcdef0123456789abcdef" // 正好 32 字节
	old, had := os.LookupEnv(userSecretSecretEnv)
	require.NoError(t, os.Setenv(userSecretSecretEnv, key))
	return func() {
		if had {
			_ = os.Setenv(userSecretSecretEnv, old)
		} else {
			_ = os.Unsetenv(userSecretSecretEnv)
		}
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	defer setTestMasterKey(t)()
	enc, err := newEncryptor()
	require.NoError(t, err)

	for _, pt := range []string{"sk-abc123", "克劳德密钥-值", "a", strings.Repeat("x", 4096)} {
		ct, err := enc.encrypt(pt)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(ct), cipherVersionPrefix), "密文需带版本前缀")
		assert.NotContains(t, string(ct), pt, "密文不得含明文")

		got, err := enc.decrypt(ct)
		require.NoError(t, err)
		assert.Equal(t, pt, got)
	}
}

func TestEncrypt_NonDeterministic(t *testing.T) {
	defer setTestMasterKey(t)()
	enc, _ := newEncryptor()
	a, _ := enc.encrypt("same")
	b, _ := enc.encrypt("same")
	assert.NotEqual(t, a, b, "随机 nonce 应使同明文密文不同")
}

func TestDecrypt_TamperFails(t *testing.T) {
	defer setTestMasterKey(t)()
	enc, _ := newEncryptor()
	ct, _ := enc.encrypt("secret")
	ct[len(ct)-1] ^= 0xFF // 篡改 tag
	_, err := enc.decrypt(ct)
	assert.Error(t, err, "GCM 认证失败应报错")
}

func TestDecrypt_BadPrefix(t *testing.T) {
	defer setTestMasterKey(t)()
	enc, _ := newEncryptor()
	_, err := enc.decrypt([]byte("plain-no-prefix"))
	assert.Error(t, err)
}

func TestNewEncryptor_MissingKey(t *testing.T) {
	old, had := os.LookupEnv(userSecretSecretEnv)
	_ = os.Unsetenv(userSecretSecretEnv)
	defer func() {
		if had {
			_ = os.Setenv(userSecretSecretEnv, old)
		}
	}()
	_, err := newEncryptor()
	assert.Error(t, err)
}

func TestMaskTail(t *testing.T) {
	assert.Equal(t, "****c123", maskTail("sk-abc123"))
	assert.Equal(t, "***", maskTail("abc"))
	assert.Equal(t, "", maskTail(""))
}
