package usersecret

import (
	"os"
	"strings"
	"testing"
)

// TestResolveNoRawErrorResponse 是源码守卫:api.go 的所有错误响应必须走统一 i18n
// 错误信封(respondErr / respondErrWithDetails → httperr.ResponseErrorLWithStatus),
// 不得出现裸 c.JSON(http.Status…)/c.AbortWith* —— 与 modules/oidc 的同款守卫一致,
// 防止 resolve 歧义这类分支再回退到绕过信封的裸响应(R1/R2 blocker 的回归闸)。
func TestResolveNoRawErrorResponse(t *testing.T) {
	data, err := os.ReadFile("api.go")
	if err != nil {
		t.Fatalf("read api.go: %v", err)
	}
	var clean strings.Builder
	for _, line := range strings.Split(string(data), "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		clean.WriteString(line)
		clean.WriteByte('\n')
	}
	cleaned := clean.String()

	for _, b := range []string{
		"c.AbortWithStatusJSON", "c.AbortWithStatus(",
		".ResponseError(", ".ResponseErrorf(", ".ResponseErrorWithStatus(",
	} {
		if strings.Contains(cleaned, b) {
			t.Fatalf("modules/usersecret/api.go must route errors through the i18n envelope, not legacy %s", b)
		}
	}
	// 裸 non-OK c.JSON(http.Status…) 同样绕过信封;成功响应走 c.Response/c.ResponseWithStatus。
	for _, line := range strings.Split(cleaned, "\n") {
		if strings.Contains(line, "c.JSON(http.Status") && !strings.Contains(line, "c.JSON(http.StatusOK") {
			t.Fatalf("modules/usersecret/api.go must not emit raw non-OK c.JSON: %s", strings.TrimSpace(line))
		}
	}
}
