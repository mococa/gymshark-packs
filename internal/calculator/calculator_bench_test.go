package calculator

import "testing"

var benchCases = []struct {
	name  string
	order int
}{
	{"Small_Order_1", 1},
	{"Small_Order_250", 250},
	{"Medium_Order_501", 501},
	{"Large_Order_12001", 12001},
	{"Very_Large_Order_50000", 50000},
	{"Edge_Case_500000", 500000},
}

var benchSizes = []int{250, 500, 1000, 2000, 5000}

// BenchmarkCalculatePacksCold measures the DP computation with no cache warmup.
// New() is excluded from timing via StopTimer/StartTimer.
func BenchmarkCalculatePacksCold(b *testing.B) {
	for _, bm := range benchCases {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				calc := New()
				b.StartTimer()
				_ = calc.CalculatePacks(bm.order, benchSizes)
			}
		})
	}
}

// BenchmarkCalculatePacksWarm measures the cache hit path.
// The first call populates the cache; b.ResetTimer excludes it from results.
func BenchmarkCalculatePacksWarm(b *testing.B) {
	for _, bm := range benchCases {
		b.Run(bm.name, func(b *testing.B) {
			calc := New()
			_ = calc.CalculatePacks(bm.order, benchSizes) // populate cache
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = calc.CalculatePacks(bm.order, benchSizes)
			}
		})
	}
}

func BenchmarkCalculatePacksArbitrarySizes(b *testing.B) {
	sizes := []int{6, 9, 20, 37, 53, 101}
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		calc := New()
		b.StartTimer()
		_ = calc.CalculatePacks(1000, sizes)
	}
}

func BenchmarkCalculatePacksManySizes(b *testing.B) {
	sizes := []int{50, 75, 100, 150, 200, 250, 300, 400, 500, 750, 1000, 1500, 2000, 3000, 5000}
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		calc := New()
		b.StartTimer()
		_ = calc.CalculatePacks(10000, sizes)
	}
}
