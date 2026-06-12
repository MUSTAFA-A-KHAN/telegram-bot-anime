package service

import "testing"

func BenchmarkNormalizeAndCompare(b *testing.B) {
	s1 := "Hello, World! This is a test string."
	s2 := "hello world this is a test string"
	for i := 0; i < b.N; i++ {
		NormalizeAndCompare(s1, s2)
	}
}
