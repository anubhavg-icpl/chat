package toc

import "strings"

// tocEscapeSet lists the bytes that must be backslash-escaped inside a quoted
// TOC argument. This is the inverse of the server's unescape function in
// server/toc, which removes a backslash before any following character.
const tocEscapeSet = `\$"{}[]()`

// escape backslash-escapes the TOC special characters in s. It is the inverse
// of the server-side unescape, so escaped text round-trips through the server
// unchanged.
func escape(s string) string {
	if !strings.ContainsAny(s, tocEscapeSet) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 4)
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(tocEscapeSet, s[i]) >= 0 {
			b.WriteByte('\\')
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// quote escapes and double-quotes s so it forms a single space-delimited TOC
// argument. This matches how the server parses commands with a space-delimited
// CSV reader.
func quote(s string) string {
	return `"` + escape(s) + `"`
}
