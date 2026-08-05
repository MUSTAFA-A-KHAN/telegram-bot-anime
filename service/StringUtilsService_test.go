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

func TestNormalizeAndComparePlural(t *testing.T) {
	tests := []struct {
		s1   string
		s2   string
		want bool
	}{
		{"apple", "apples", true},
		{"apples", "apple", true},
		{"box", "boxes", true},
		{"boxes", "box", true},
		{"city", "cities", false}, // We don't handle complex plurals right now per simple spec
		{"apple", "orange", false},
		{"car", "cars", true},
		{"cars", "car", true},
		{"dog", "doges", true}, // "dog" + "es" = "doges", but wait: doges vs dog, len=5 vs 3, diff=2. "es" matches. Wait, is "doges" a plural of "dog"? No, "dogs" is. But simple rule allows it. That's fine per spec.
		{"match", "matches", true},
		{"matches", "match", true},
		{"apple", "apple", true},
		{"", "", true},
		{"a", "as", true},
	}

	for _, tt := range tests {
		if got := NormalizeAndComparePlural(tt.s1, tt.s2); got != tt.want {
			t.Errorf("NormalizeAndComparePlural(%q, %q) = %v; want %v", tt.s1, tt.s2, got, tt.want)
		}
	}
}
