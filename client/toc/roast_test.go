package toc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRoastPassword verifies RoastPassword against vectors derived from the
// server's wire.RoastTOCPassword, which XORs with the same "Tic/Toc" table. The
// "input->want" pairs are computed as input XOR table so they match the server
// exactly (see server/toc/cmd_client_test.go which feeds these through the
// server's toc_signon path).
func TestRoastPassword(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  []byte
	}{
		{
			name:  "empty password",
			input: []byte{},
			want:  []byte{},
		},
		{
			name:  "single byte equal to first key byte",
			input: []byte{0x54}, // 'T'
			want:  []byte{0x00}, // 0x54 ^ 'T'
		},
		{
			name:  "password equal to full key is all zeros",
			input: []byte("Tic/Toc"),
			want:  []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:  "password longer than the key wraps",
			input: []byte("Tic/TocTic/Toc"),
			want:  []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:  "clear password maps to known roasted bytes",
			input: []byte("password"),
			want:  []byte{0x24, 0x08, 0x10, 0x5C, 0x23, 0x00, 0x11, 0x30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RoastPassword(tt.input))
		})
	}
}

// TestRoastPassword_Reversible confirms roasting is symmetric: roasting a
// roasted value yields the original clear text.
func TestRoastPassword_Reversible(t *testing.T) {
	for _, clear := range []string{"", "a", "password", "Tic/Toc", "a much longer password than the key"} {
		roasted := RoastPassword([]byte(clear))
		assert.Equal(t, []byte(clear), RoastPassword(roasted), "round trip for %q", clear)
	}
}
