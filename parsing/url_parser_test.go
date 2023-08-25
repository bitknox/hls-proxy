package parsing

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInputUrl(t *testing.T) {

	r, err := ParseInputUrl(base64.StdEncoding.EncodeToString([]byte("a|b|c")))
	if err != nil {
		t.Fatal("Error parsing base64 string")
	}
	assert.Equal(t, "a", r.Url)
	assert.Equal(t, "b", r.Referer)
	assert.Equal(t, "c", r.Origin)

	r, err = ParseInputUrl(base64.StdEncoding.EncodeToString([]byte("a?|b")))
	if err != nil {
		t.Fatal("Error parsing base64 string")
	}
	assert.Equal(t, "a", r.Url)
	assert.Equal(t, "b", r.Referer)
	assert.Equal(t, "", r.Origin)

	r, err = ParseInputUrl(base64.StdEncoding.EncodeToString([]byte("a")))
	if err != nil {
		t.Fatal("Error parsing base64 string")
	}
	assert.Equal(t, "a", r.Url)
	assert.Equal(t, "", r.Referer)
	assert.Equal(t, "", r.Origin)

}
