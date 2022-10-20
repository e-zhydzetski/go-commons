package test

import (
	"testing"
)

func BenchmarkMap(b *testing.B) {
	const amount = 100000
	test := func(mFactory func() map[int]int) func(b *testing.B) {
		return func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				m := mFactory()
				for i := 0; i < amount; i++ {
					m[i] = i
				}
			}
		}
	}

	b.Run("Simple", test(func() map[int]int {
		return map[int]int{}
	}))
	b.Run("WithHint", test(func() map[int]int {
		return make(map[int]int, amount)
	}))
}
