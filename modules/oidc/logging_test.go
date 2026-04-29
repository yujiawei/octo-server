package oidc

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubHash_DeterministicShortHex(t *testing.T) {
	a := subHash("user-12345")
	b := subHash("user-12345")
	assert.Equal(t, a, b, "same input must produce same hash")
	assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{8}$`), a)
}

func TestSubHash_DifferentInputsDifferentOutputs(t *testing.T) {
	a := subHash("user-1")
	b := subHash("user-2")
	assert.NotEqual(t, a, b)
}

func TestSubHash_EmptyReturnsEmpty(t *testing.T) {
	assert.Equal(t, "", subHash(""))
}

func TestNewTraceID_FormatAndUniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		tid := newTraceID()
		assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{16}$`), tid)
		_, dup := seen[tid]
		assert.False(t, dup, "trace_id collisions are unexpected for 100 samples")
		seen[tid] = struct{}{}
	}
}
