package fastcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXXHashString(t *testing.T) {
	hash := xxHashString("1")
	hash = xxHashString("11111111111111111111111111111111111111111111111111111")
	assert.Greater(t, hash, uint64(0))
}
