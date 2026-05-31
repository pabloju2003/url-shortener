package shortener

import (
	"strings"
	"testing"
)

func TestGenerate_Length(t *testing.T) {
	for _, length := range []int{1, 6, 10, 32} {
		code, err := Generate(length)
		if err != nil {
			t.Fatalf("Generate(%d) returned error: %v", length, err)
		}
		if len(code) != length {
			t.Errorf("Generate(%d) = %q, want length %d", length, code, length)
		}
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := range 1000 {
		code, err := Generate(8)
		if err != nil {
			t.Fatalf("Generate failed on iteration %d: %v", i, err)
		}
		if _, dup := seen[code]; dup {
			t.Fatalf("duplicate code %q after %d iterations", code, i)
		}
		seen[code] = struct{}{}
	}
}

func TestGenerate_ValidChars(t *testing.T) {
	for range 100 {
		code, err := Generate(10)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}
		for _, c := range code {
			if !strings.ContainsRune(charset, c) {
				t.Errorf("code %q contains invalid character %q", code, c)
			}
		}
	}
}

func TestIsValid(t *testing.T) {
	cases := []struct {
		code  string
		valid bool
	}{
		{"abc123", true},
		{"AbCdEf", true},
		{"ZZZZZZ", true},
		{"aB3xY9", true},
		{"", false},
		{"abc 123", false},
		{"abc-123", false},
		{"abc_123", false},
		{"héllo", false},
		{"abc\n123", false},
	}
	for _, tc := range cases {
		if got := IsValid(tc.code); got != tc.valid {
			t.Errorf("IsValid(%q) = %v, want %v", tc.code, got, tc.valid)
		}
	}
}
