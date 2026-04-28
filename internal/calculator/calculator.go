package calculator

import (
	"fmt"
	"math"
	"sort"
)

// Calculator computes optimal pack breakdowns and caches results internally.
type Calculator struct {
	cache *packCache
}

// New returns a Calculator with a bounded result cache.
func New() *Calculator {
	return &Calculator{cache: newPackCache(maxCacheEntries)}
}

// CalculatePacks returns the optimal pack breakdown for a given order quantity.
// Priority: 1) minimize total items shipped, 2) minimize number of packs.
func (c *Calculator) CalculatePacks(order int, sizes []int) map[int]int {
	if order <= 0 || len(sizes) == 0 {
		return map[int]int{}
	}

	sortedSizes := make([]int, len(sizes))
	copy(sortedSizes, sizes)
	sort.Sort(sort.Reverse(sort.IntSlice(sortedSizes)))

	key := fmt.Sprintf("%d|%v", order, sortedSizes)
	if cached, ok := c.cache.get(key); ok {
		return cached
	}

	largest := sortedSizes[0]
	maxSearch := order + largest

	dp := make([]int, maxSearch+1)
	from := make([]int, maxSearch+1)
	for i := range dp {
		dp[i] = math.MaxInt
	}
	dp[0] = 0

	for i := 1; i <= maxSearch; i++ {
		for _, packSize := range sortedSizes {
			if packSize <= i && dp[i-packSize] != math.MaxInt {
				if dp[i-packSize]+1 < dp[i] {
					dp[i] = dp[i-packSize] + 1
					from[i] = packSize
				}
			}
		}
	}

	best := -1
	for i := order; i <= maxSearch; i++ {
		if dp[i] != math.MaxInt {
			best = i
			break
		}
	}

	if best == -1 {
		return map[int]int{}
	}

	result := map[int]int{}
	for cur := best; cur > 0; {
		packSize := from[cur]
		result[packSize]++
		cur -= packSize
	}

	c.cache.set(key, result)
	return result
}

// TotalItems returns the total number of items across all packs.
func TotalItems(packs map[int]int) int {
	total := 0
	for size, count := range packs {
		total += size * count
	}
	return total
}

// TotalPacks returns the total number of packs.
func TotalPacks(packs map[int]int) int {
	total := 0
	for _, count := range packs {
		total += count
	}
	return total
}
