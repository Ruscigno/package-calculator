package algorithm

import (
	"testing"
)

func TestCalculate_Standard(t *testing.T) {
	tests := []struct {
		name      string
		order     int
		packSizes []int
		wantPacks map[int]int
		wantItems int
		wantTotal int
		wantWaste int
	}{
		{
			name:      "Example: 251 items",
			order:     251,
			packSizes: []int{250, 500, 1000, 2000, 5000},
			wantPacks: map[int]int{500: 1},
			wantItems: 500,
			wantTotal: 1,
			wantWaste: 249,
		},
		{
			name:      "Exact match: 250 items",
			order:     250,
			packSizes: []int{250, 500, 1000, 2000, 5000},
			wantPacks: map[int]int{250: 1},
			wantItems: 250,
			wantTotal: 1,
			wantWaste: 0,
		},
		{
			name:      "Multiple packs: 501 items",
			order:     501,
			packSizes: []int{250, 500, 1000, 2000, 5000},
			wantPacks: map[int]int{250: 1, 500: 1},
			wantItems: 750,
			wantTotal: 2,
			wantWaste: 249,
		},
		{
			name:      "Large order: 12001 items",
			order:     12001,
			packSizes: []int{250, 500, 1000, 2000, 5000},
			wantPacks: map[int]int{250: 1, 2000: 1, 5000: 2},
			wantItems: 12250,
			wantTotal: 4,
			wantWaste: 249,
		},
		{
			name:      "Single pack size",
			order:     100,
			packSizes: []int{50},
			wantPacks: map[int]int{50: 2},
			wantItems: 100,
			wantTotal: 2,
			wantWaste: 0,
		},
		{
			name:      "Small order with large packs",
			order:     1,
			packSizes: []int{250, 500, 1000},
			wantPacks: map[int]int{250: 1},
			wantItems: 250,
			wantTotal: 1,
			wantWaste: 249,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Calculate(tt.order, tt.packSizes)

			if result.TotalItems != tt.wantItems {
				t.Errorf("TotalItems = %d, want %d", result.TotalItems, tt.wantItems)
			}

			if result.TotalPacks != tt.wantTotal {
				t.Errorf("TotalPacks = %d, want %d", result.TotalPacks, tt.wantTotal)
			}

			if result.Waste != tt.wantWaste {
				t.Errorf("Waste = %d, want %d", result.Waste, tt.wantWaste)
			}

			// Check pack counts
			if len(result.PackCounts) != len(tt.wantPacks) {
				t.Errorf("PackCounts length = %d, want %d", len(result.PackCounts), len(tt.wantPacks))
			}

			for size, count := range tt.wantPacks {
				if result.PackCounts[size] != count {
					t.Errorf("PackCounts[%d] = %d, want %d", size, result.PackCounts[size], count)
				}
			}
		})
	}
}

func TestCalculate_EdgeCase(t *testing.T) {
	// The critical edge case from the requirements
	order := 500000
	packSizes := []int{23, 31, 53}

	result := Calculate(order, packSizes)

	// Expected: {23: 2, 31: 7, 53: 9429}
	// Total: 23*2 + 31*7 + 53*9429 = 46 + 217 + 499737 = 500000
	expected := map[int]int{23: 2, 31: 7, 53: 9429}

	if result.TotalItems != 500000 {
		t.Errorf("TotalItems = %d, want 500000", result.TotalItems)
	}

	if result.Waste != 0 {
		t.Errorf("Waste = %d, want 0", result.Waste)
	}

	// Verify the pack counts
	for size, count := range expected {
		if result.PackCounts[size] != count {
			t.Errorf("PackCounts[%d] = %d, want %d", size, result.PackCounts[size], count)
		}
	}

	// Verify total calculation
	total := 0
	for size, count := range result.PackCounts {
		total += size * count
	}
	if total != 500000 {
		t.Errorf("Calculated total = %d, want 500000", total)
	}

	t.Logf("Edge case result: %+v", result.PackCounts)
}

func TestCalculate_EmptyInput(t *testing.T) {
	tests := []struct {
		name      string
		order     int
		packSizes []int
	}{
		{
			name:      "Zero order",
			order:     0,
			packSizes: []int{250, 500},
		},
		{
			name:      "Negative order",
			order:     -100,
			packSizes: []int{250, 500},
		},
		{
			name:      "Empty pack sizes",
			order:     100,
			packSizes: []int{},
		},
		{
			name:      "Nil pack sizes",
			order:     100,
			packSizes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Calculate(tt.order, tt.packSizes)

			if len(result.PackCounts) != 0 {
				t.Errorf("Expected empty result, got %+v", result.PackCounts)
			}
		})
	}
}

func TestCalculate_PrimePackSizes(t *testing.T) {
	// Test with prime numbers which are harder to optimize
	order := 100
	packSizes := []int{7, 11, 13}

	result := Calculate(order, packSizes)

	// Should find a valid solution
	if result.TotalItems < order {
		t.Errorf("TotalItems = %d, want >= %d", result.TotalItems, order)
	}

	// Verify the calculation
	total := 0
	for size, count := range result.PackCounts {
		total += size * count
	}
	if total != result.TotalItems {
		t.Errorf("Calculated total = %d, want %d", total, result.TotalItems)
	}

	t.Logf("Prime pack sizes result for order %d: %+v (total: %d, packs: %d)",
		order, result.PackCounts, result.TotalItems, result.TotalPacks)
}

func TestCalculate_MinimizeItems(t *testing.T) {
	// Test that Rule 2 (minimize items) takes precedence over Rule 3 (minimize packs)
	order := 10
	packSizes := []int{3, 5}

	result := Calculate(order, packSizes)

	// Should prefer 5+5=10 (2 packs) over 3+3+3+3=12 (4 packs)
	// because 10 < 12 (fewer items is more important)
	if result.TotalItems != 10 {
		t.Errorf("TotalItems = %d, want 10 (Rule 2: minimize items)", result.TotalItems)
	}

	expected := map[int]int{5: 2}
	for size, count := range expected {
		if result.PackCounts[size] != count {
			t.Errorf("PackCounts[%d] = %d, want %d", size, result.PackCounts[size], count)
		}
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		packSizes []int
		want      bool
	}{
		{
			name:      "Valid pack sizes",
			packSizes: []int{250, 500, 1000},
			want:      true,
		},
		{
			name:      "Empty pack sizes",
			packSizes: []int{},
			want:      false,
		},
		{
			name:      "Zero in pack sizes",
			packSizes: []int{0, 250, 500},
			want:      false,
		},
		{
			name:      "Negative in pack sizes",
			packSizes: []int{-100, 250, 500},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Validate(tt.packSizes)
			if got != tt.want {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Benchmark tests
func BenchmarkCalculate_Small(b *testing.B) {
	packSizes := []int{250, 500, 1000, 2000, 5000}
	order := 251

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Calculate(order, packSizes)
	}
}

func BenchmarkCalculate_Medium(b *testing.B) {
	packSizes := []int{250, 500, 1000, 2000, 5000}
	order := 12001

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Calculate(order, packSizes)
	}
}

func BenchmarkCalculate_EdgeCase(b *testing.B) {
	packSizes := []int{23, 31, 53}
	order := 500000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Calculate(order, packSizes)
	}
}

// ============================================================================
// COMPREHENSIVE EDGE CASE TESTS FROM EDGE_CASES.md
// ============================================================================

// Edge Case #2: Impossible Exact Match (Rule 2 Triggers Overfill)
func TestEdgeCase_ImpossibleExactMatch(t *testing.T) {
	order := 4
	packSizes := []int{3, 5}

	result := Calculate(order, packSizes)

	// No combination makes exactly 4
	// Possible sums: 0, 3, 5, 6, 8, 9, 10, ...
	// Best: 1 × 5 → 5 items (waste: 1)
	// NOT: 2 × 3 → 6 items (waste: 2) - more waste

	if result.TotalItems != 5 {
		t.Errorf("TotalItems = %d, want 5 (least items >= 4)", result.TotalItems)
	}

	if result.Waste != 1 {
		t.Errorf("Waste = %d, want 1", result.Waste)
	}

	expected := map[int]int{5: 1}
	if len(result.PackCounts) != len(expected) {
		t.Errorf("PackCounts = %+v, want %+v", result.PackCounts, expected)
	}

	for size, count := range expected {
		if result.PackCounts[size] != count {
			t.Errorf("PackCounts[%d] = %d, want %d", size, result.PackCounts[size], count)
		}
	}

	t.Logf("✅ Edge Case #2: Impossible exact match - Result: %+v", result.PackCounts)
}

// Edge Case #3: Multiple Paths to Same Item Count (Rule 3 Triggers)
func TestEdgeCase_MultiplePathsFewestPacks(t *testing.T) {
	order := 6
	packSizes := []int{1, 2, 3}

	result := Calculate(order, packSizes)

	// Multiple ways to reach 6:
	// - 2 × 3 = 6 (2 packs) ✓ BEST
	// - 3 × 2 = 6 (3 packs)
	// - 6 × 1 = 6 (6 packs)
	// - 1 × 3 + 3 × 1 = 6 (4 packs)

	if result.TotalItems != 6 {
		t.Errorf("TotalItems = %d, want 6", result.TotalItems)
	}

	if result.TotalPacks != 2 {
		t.Errorf("TotalPacks = %d, want 2 (fewest packs for 6 items)", result.TotalPacks)
	}

	if result.Waste != 0 {
		t.Errorf("Waste = %d, want 0", result.Waste)
	}

	expected := map[int]int{3: 2}
	for size, count := range expected {
		if result.PackCounts[size] != count {
			t.Errorf("PackCounts[%d] = %d, want %d", size, result.PackCounts[size], count)
		}
	}

	t.Logf("✅ Edge Case #3: Multiple paths - Result: %+v (Rule 3: fewest packs)", result.PackCounts)
}

// Edge Case #4: Single Item Orders (Minimal Case)
func TestEdgeCase_SingleItemOrder(t *testing.T) {
	order := 1
	packSizes := []int{250, 500, 1000, 2000, 5000}

	result := Calculate(order, packSizes)

	// Must choose smallest pack that covers 1 item
	// Correct: 1 × 250
	// Incorrect: 1 × 500 (violates Rule 2 - more items)

	if result.TotalItems != 250 {
		t.Errorf("TotalItems = %d, want 250 (smallest pack)", result.TotalItems)
	}

	expected := map[int]int{250: 1}
	for size, count := range expected {
		if result.PackCounts[size] != count {
			t.Errorf("PackCounts[%d] = %d, want %d", size, result.PackCounts[size], count)
		}
	}

	t.Logf("✅ Edge Case #4: Single item order - Result: %+v", result.PackCounts)
}

// Edge Case #6: Prime Pack Sizes (Hard to Combine)
func TestEdgeCase_PrimePackSizes(t *testing.T) {
	order := 100
	packSizes := []int{7, 13, 17}

	result := Calculate(order, packSizes)

	// No small common factors
	// Forces algorithm to explore many combinations
	// Must not hang or OOM

	if result.TotalItems < order {
		t.Errorf("TotalItems = %d, want >= %d", result.TotalItems, order)
	}

	// Verify calculation
	total := 0
	for size, count := range result.PackCounts {
		total += size * count
	}
	if total != result.TotalItems {
		t.Errorf("Calculated total = %d, want %d", total, result.TotalItems)
	}

	t.Logf("✅ Edge Case #6: Prime pack sizes - Result: %+v (total: %d, packs: %d)",
		result.PackCounts, result.TotalItems, result.TotalPacks)
}

// Edge Case #8: Pack Size Larger Than Order
func TestEdgeCase_PackSizeLargerThanOrder(t *testing.T) {
	order := 500
	packSizes := []int{1000, 2000}

	result := Calculate(order, packSizes)

	// Don't skip large packs just because they're > order
	// Must still consider them
	// Expected: 1 × 1000

	if result.TotalItems != 1000 {
		t.Errorf("TotalItems = %d, want 1000 (smallest pack even though > order)", result.TotalItems)
	}

	expected := map[int]int{1000: 1}
	for size, count := range expected {
		if result.PackCounts[size] != count {
			t.Errorf("PackCounts[%d] = %d, want %d", size, result.PackCounts[size], count)
		}
	}

	t.Logf("✅ Edge Case #8: Pack larger than order - Result: %+v", result.PackCounts)
}

// Edge Case #9: Duplicate Pack Sizes
func TestEdgeCase_DuplicatePackSizes(t *testing.T) {
	order := 500
	packSizes := []int{250, 500, 500, 1000}

	result := Calculate(order, packSizes)

	// Should handle duplicates gracefully
	// Expected: 1 × 500 (exact match)

	if result.TotalItems != 500 {
		t.Errorf("TotalItems = %d, want 500", result.TotalItems)
	}

	if result.Waste != 0 {
		t.Errorf("Waste = %d, want 0", result.Waste)
	}

	// Should have 1 pack of size 500
	if result.PackCounts[500] != 1 {
		t.Errorf("PackCounts[500] = %d, want 1", result.PackCounts[500])
	}

	t.Logf("✅ Edge Case #9: Duplicate pack sizes - Result: %+v", result.PackCounts)
}

// Comprehensive Edge Case Test Suite (Summary)
func TestEdgeCases_ComprehensiveSuite(t *testing.T) {
	cases := []struct {
		name      string
		sizes     []int
		order     int
		expected  map[int]int
		wantItems int
		wantPacks int
	}{
		{
			name:      "Given Large Edge Case",
			sizes:     []int{23, 31, 53},
			order:     500000,
			expected:  map[int]int{23: 2, 31: 7, 53: 9429},
			wantItems: 500000,
			wantPacks: 9438,
		},
		{
			name:      "No Exact Match",
			sizes:     []int{3, 5},
			order:     4,
			expected:  map[int]int{5: 1},
			wantItems: 5,
			wantPacks: 1,
		},
		{
			name:      "Fewest Packs",
			sizes:     []int{1, 2, 3},
			order:     6,
			expected:  map[int]int{3: 2},
			wantItems: 6,
			wantPacks: 2,
		},
		{
			name:      "Single Item",
			sizes:     []int{250, 500},
			order:     1,
			expected:  map[int]int{250: 1},
			wantItems: 250,
			wantPacks: 1,
		},
		{
			name:      "Large Pack",
			sizes:     []int{1000},
			order:     500,
			expected:  map[int]int{1000: 1},
			wantItems: 1000,
			wantPacks: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := Calculate(tc.order, tc.sizes)

			if result.TotalItems != tc.wantItems {
				t.Errorf("TotalItems = %d, want %d", result.TotalItems, tc.wantItems)
			}

			if result.TotalPacks != tc.wantPacks {
				t.Errorf("TotalPacks = %d, want %d", result.TotalPacks, tc.wantPacks)
			}

			for size, count := range tc.expected {
				if result.PackCounts[size] != count {
					t.Errorf("PackCounts[%d] = %d, want %d", size, result.PackCounts[size], count)
				}
			}

			t.Logf("✅ %s: %+v", tc.name, result.PackCounts)
		})
	}
}
