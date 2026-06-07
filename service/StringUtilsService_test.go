package service

import "testing"

func TestNormalizeAndCompare(t *testing.T) {
	tests := []struct {
		s1   string
		s2   string
		want bool
	}{
		{"Hello, World!  This is a test.", "hello world this is a test", true},
		{"Foo, Bar!", "foo bar", true},
		{"Foo,  Bar!  ", "foo bar", true},
		{"  Foo,  Bar!  ", "foo bar", true},
		{"test-word", "testword", true},
		{"A B C", "a b c", true},
		{"Hello_world 123 !@#", "hello_world 123", true},
		{" \t\n whitespace \t\n ", "whitespace", true},
		{"", "", true},
		{"!", "", true},
	}

	for _, tt := range tests {
		if got := NormalizeAndCompare(tt.s1, tt.s2); got != tt.want {
			t.Errorf("NormalizeAndCompare(%q, %q) = %v; want %v", tt.s1, tt.s2, got, tt.want)
		}
	}
}
