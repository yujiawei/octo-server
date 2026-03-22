package space

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mininglamp-OSS/octo-lib/pkg/wkhttp"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// wrapWK converts a wkhttp.HandlerFunc into a gin.HandlerFunc for testing.
func wrapWK(h wkhttp.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		wc := &wkhttp.Context{Context: c}
		h(wc)
	}
}

var testCache *InMemoryMembershipCache

// setupRouter creates a test router with the space middleware and a simple 200 handler.
func setupRouter(checker MembershipChecker) *gin.Engine {
	testCache = NewInMemoryMembershipCache()
	r := gin.New()
	mw := spaceMiddleware(checker, testCache)
	r.Use(func(c *gin.Context) {
		// simulate auth: set uid so GetLoginUID works
		c.Set("uid", "testuser")
		c.Set("name", "Test")
		c.Next()
	})
	r.Use(wrapWK(mw))
	r.GET("/test", func(c *gin.Context) {
		spaceID, _ := c.Get("space_id")
		c.JSON(http.StatusOK, gin.H{"space_id": spaceID})
	})
	return r
}

func TestSpaceMiddleware_NoSpaceID_PassThrough(t *testing.T) {
	called := false
	checker := func(spaceID, uid string) (bool, error) {
		called = true
		return false, nil
	}
	r := setupRouter(checker)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, called, "checker should not be called when no space_id")
}

func TestSpaceMiddleware_NotMember_403(t *testing.T) {
	checker := func(spaceID, uid string) (bool, error) {
		return false, nil
	}
	r := setupRouter(checker)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?space_id=sp1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSpaceMiddleware_IsMember_PassWithContext(t *testing.T) {
	checker := func(spaceID, uid string) (bool, error) {
		return true, nil
	}
	r := setupRouter(checker)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?space_id=sp1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "sp1")
}

func TestSpaceMiddleware_Header_SpaceID(t *testing.T) {
	checker := func(spaceID, uid string) (bool, error) {
		assert.Equal(t, "sp-header", spaceID)
		return true, nil
	}
	r := setupRouter(checker)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Space-ID", "sp-header")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSpaceMiddleware_CacheHit(t *testing.T) {
	callCount := 0
	checker := func(spaceID, uid string) (bool, error) {
		callCount++
		return true, nil
	}
	r := setupRouter(checker)

	// first request — cache miss
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?space_id=sp1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, callCount)

	// second request — cache hit
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test?space_id=sp1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, callCount, "checker should not be called again due to cache")
}
