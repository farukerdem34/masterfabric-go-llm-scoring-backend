package middleware

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORSOptions_WildcardAcceptsAnyOrigin(t *testing.T) {
	opts := CORSOptions([]string{"*"})
	assert.True(t, opts.AllowCredentials)
	assert.NotNil(t, opts.AllowOriginFunc)

	assert.True(t, opts.AllowOriginFunc(nil, "http://localhost:3000"))
	assert.True(t, opts.AllowOriginFunc(nil, "https://any-origin.com"))
	assert.False(t, opts.AllowOriginFunc(nil, ""))
}

func TestCORSOptions_EmptyDeniesAll(t *testing.T) {
	opts := CORSOptions(nil)
	assert.False(t, opts.AllowCredentials)
	assert.NotNil(t, opts.AllowOriginFunc)

	assert.False(t, opts.AllowOriginFunc(nil, "http://localhost:3000"))
	assert.False(t, opts.AllowOriginFunc(nil, "https://example.com"))
}

func TestCORSOptions_EmptySliceDeniesAll(t *testing.T) {
	opts := CORSOptions([]string{})
	assert.False(t, opts.AllowCredentials)
	assert.False(t, opts.AllowOriginFunc(nil, "http://localhost:3000"))
}

func TestCORSOptions_ExplicitOriginsMatchAndReject(t *testing.T) {
	opts := CORSOptions([]string{"http://localhost:3000", "https://app.example.com"})
	assert.True(t, opts.AllowCredentials)
	assert.NotNil(t, opts.AllowOriginFunc)

	r := &http.Request{}
	assert.True(t, opts.AllowOriginFunc(r, "http://localhost:3000"))
	assert.True(t, opts.AllowOriginFunc(r, "https://app.example.com"))
	assert.False(t, opts.AllowOriginFunc(r, "https://evil.com"))
	assert.False(t, opts.AllowOriginFunc(r, ""))
}


