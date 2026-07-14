package toc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscape(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain text unchanged", in: "hello world", want: "hello world"},
		{name: "empty", in: "", want: ""},
		{name: "escapes parentheses", in: "hi :)", want: `hi :\)`},
		{name: "escapes all special chars", in: `"$ {}[]()`, want: `\"\$ \{\}\[\]\(\)`},
		{name: "escapes backslash", in: `a\b`, want: `a\\b`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, escape(tt.in))
		})
	}
}

func TestQuote(t *testing.T) {
	assert.Equal(t, `"hi there :\)"`, quote("hi there :)"))
	assert.Equal(t, `""`, quote(""))
}
