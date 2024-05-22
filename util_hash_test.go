package fastcache

import "testing"

func TestXXHashString(t *testing.T) {
	hash := xxHashString("1")
	t.Log(hash)
	hash = xxHashString("11111111111111111111111111111111111111111111111111111")
	t.Log(hash)
}
