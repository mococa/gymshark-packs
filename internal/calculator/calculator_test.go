package calculator

import (
	"reflect"
	"testing"
)

func TestCalculatePacks(t *testing.T) {
	calc := New()
	defaultSizes := []int{250, 500, 1000, 2000, 5000}

	tests := []struct {
		name          string
		order         int
		sizes         []int
		expectedPacks map[int]int
		expectedItems int
		expectedCount int
	}{
		{
			name:          "order 1 item",
			order:         1,
			sizes:         defaultSizes,
			expectedPacks: map[int]int{250: 1},
			expectedItems: 250,
			expectedCount: 1,
		},
		{
			name:          "order exactly 250",
			order:         250,
			sizes:         defaultSizes,
			expectedPacks: map[int]int{250: 1},
			expectedItems: 250,
			expectedCount: 1,
		},
		{
			name:          "order 251 items",
			order:         251,
			sizes:         defaultSizes,
			expectedPacks: map[int]int{500: 1},
			expectedItems: 500,
			expectedCount: 1,
		},
		{
			name:          "order 501 items",
			order:         501,
			sizes:         defaultSizes,
			expectedPacks: map[int]int{250: 1, 500: 1},
			expectedItems: 750,
			expectedCount: 2,
		},
		{
			name:          "order 12001 items",
			order:         12001,
			sizes:         defaultSizes,
			expectedPacks: map[int]int{250: 1, 2000: 1, 5000: 2},
			expectedItems: 12250,
			expectedCount: 4,
		},
		{
			name:          "order exactly 5000",
			order:         5000,
			sizes:         defaultSizes,
			expectedPacks: map[int]int{5000: 1},
			expectedItems: 5000,
			expectedCount: 1,
		},
		{
			name:          "order 5001",
			order:         5001,
			sizes:         defaultSizes,
			expectedPacks: map[int]int{5000: 1, 250: 1},
			expectedItems: 5250,
			expectedCount: 2,
		},
		{
			name:          "order 2500",
			order:         2500,
			sizes:         defaultSizes,
			expectedPacks: map[int]int{2000: 1, 500: 1},
			expectedItems: 2500,
			expectedCount: 2,
		},
		{
			name:          "order 2501",
			order:         2501,
			sizes:         defaultSizes,
			expectedPacks: map[int]int{2000: 1, 500: 1, 250: 1},
			expectedItems: 2750,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CalculatePacks(tt.order, tt.sizes)

			// Check pack breakdown matches
			if !reflect.DeepEqual(result, tt.expectedPacks) {
				t.Errorf("CalculatePacks(%d) packs = %v, want %v", tt.order, result, tt.expectedPacks)
			}

			// Verify total items
			totalItems := TotalItems(result)
			if totalItems != tt.expectedItems {
				t.Errorf("TotalItems = %d, want %d", totalItems, tt.expectedItems)
			}

			// Verify total packs
			totalPacks := TotalPacks(result)
			if totalPacks != tt.expectedCount {
				t.Errorf("TotalPacks = %d, want %d", totalPacks, tt.expectedCount)
			}

			// Verify we meet or exceed the order
			if totalItems < tt.order {
				t.Errorf("TotalItems %d is less than order %d", totalItems, tt.order)
			}
		})
	}
}

func TestCalculatePacksEdgeCases(t *testing.T) {
	calc := New()

	t.Run("zero order", func(t *testing.T) {
		result := calc.CalculatePacks(0, []int{250, 500})
		if len(result) != 0 {
			t.Errorf("CalculatePacks(0) = %v, want empty map", result)
		}
	})

	t.Run("negative order", func(t *testing.T) {
		result := calc.CalculatePacks(-100, []int{250, 500})
		if len(result) != 0 {
			t.Errorf("CalculatePacks(-100) = %v, want empty map", result)
		}
	})

	t.Run("empty sizes", func(t *testing.T) {
		result := calc.CalculatePacks(100, []int{})
		if len(result) != 0 {
			t.Errorf("CalculatePacks with empty sizes = %v, want empty map", result)
		}
	})

	t.Run("single pack size", func(t *testing.T) {
		result := calc.CalculatePacks(100, []int{50})
		expected := map[int]int{50: 2}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("CalculatePacks(100, [50]) = %v, want %v", result, expected)
		}
	})
}

func TestCalculatePacksArbitrarySizes(t *testing.T) {
	calc := New()

	t.Run("arbitrary sizes - greedy would fail", func(t *testing.T) {
		// With sizes [6, 9, 20], order 15:
		// Greedy (largest first): 20 (total 20, overage 5)
		// DP optimal: 9 + 6 = 15 (zero overage)
		result := calc.CalculatePacks(15, []int{6, 9, 20})
		expected := map[int]int{6: 1, 9: 1}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("CalculatePacks(15, [6,9,20]) = %v, want %v", result, expected)
		}
		if total := TotalItems(result); total != 15 {
			t.Errorf("TotalItems = %d, want 15", total)
		}
	})

	t.Run("prime numbers", func(t *testing.T) {
		result := calc.CalculatePacks(100, []int{7, 11, 13})
		total := TotalItems(result)
		if total < 100 {
			t.Errorf("TotalItems %d is less than order 100", total)
		}
	})

	// Edge case from RE Partners email
	t.Run("edge case: sizes [23,31,53] order 500000", func(t *testing.T) {
		sizes := []int{23, 31, 53}
		order := 500000
		expected := map[int]int{23: 2, 31: 7, 53: 9429}

		result := calc.CalculatePacks(order, sizes)

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("CalculatePacks(%d, %v) = %v, want %v", order, sizes, result, expected)
		}

		total := TotalItems(result)
		expectedTotal := 23*2 + 31*7 + 53*9429 // = 500000 exactly
		if total != expectedTotal {
			t.Errorf("TotalItems = %d, want %d", total, expectedTotal)
		}
	})
}

func TestTotalItems(t *testing.T) {
	packs := map[int]int{
		250:  1,
		500:  2,
		1000: 3,
	}
	expected := 250 + 1000 + 3000
	result := TotalItems(packs)
	if result != expected {
		t.Errorf("TotalItems(%v) = %d, want %d", packs, result, expected)
	}
}

func TestTotalPacks(t *testing.T) {
	packs := map[int]int{
		250:  1,
		500:  2,
		1000: 3,
	}
	expected := 6
	result := TotalPacks(packs)
	if result != expected {
		t.Errorf("TotalPacks(%v) = %d, want %d", packs, result, expected)
	}
}
