package algorithm

import (
	"math"
	"sort"
)

// Result represents the calculation result
type Result struct {
	PackCounts map[int]int // pack size -> count
	TotalItems int         // total items delivered
	TotalPacks int         // total number of packs
	Waste      int         // items - order
}

// Calculate finds the optimal pack combination for a given order quantity.
// It follows these rules in order of priority:
// 1. Only whole packs can be sent
// 2. Minimize total items (must be >= order)
// 3. Among solutions with same total items, minimize number of packs
//
// Algorithm: Dynamic Programming with backtracking
// Time Complexity: O(order * len(packSizes))
// Space Complexity: O(order)
func Calculate(order int, packSizes []int) Result {
	if order <= 0 {
		return Result{PackCounts: make(map[int]int)}
	}

	if len(packSizes) == 0 {
		return Result{PackCounts: make(map[int]int)}
	}

	// Sort pack sizes ascending for DP
	sizes := make([]int, len(packSizes))
	copy(sizes, packSizes)
	sort.Ints(sizes)

	// Find the maximum pack size for upper bound
	maxSize := sizes[len(sizes)-1]

	// We need to check up to order + maxSize to find solutions that exceed order
	limit := order + maxSize

	// dp[i] = minimum number of packs to make exactly i items
	// Initialize with infinity (impossible)
	dp := make([]int, limit+1)
	for i := range dp {
		dp[i] = math.MaxInt32
	}
	dp[0] = 0

	// parent[i] = the pack size used to reach i items optimally
	parent := make([]int, limit+1)

	// Fill DP table
	for i := 1; i <= limit; i++ {
		for _, size := range sizes {
			if size <= i && dp[i-size] != math.MaxInt32 {
				if dp[i-size]+1 < dp[i] {
					dp[i] = dp[i-size] + 1
					parent[i] = size
				}
			}
		}
	}

	// Find the minimum items >= order with fewest packs
	// Rule 2: Minimize items first
	// Rule 3: Among those, minimize packs
	bestItems := -1
	bestPacks := math.MaxInt32

	for items := order; items <= limit; items++ {
		if dp[items] != math.MaxInt32 {
			// Found a valid solution
			if bestItems == -1 {
				// First valid solution
				bestItems = items
				bestPacks = dp[items]
			} else if items < bestItems {
				// Better solution (fewer items)
				bestItems = items
				bestPacks = dp[items]
			} else if items == bestItems && dp[items] < bestPacks {
				// Same items but fewer packs
				bestPacks = dp[items]
			} else if items > bestItems {
				// We've found the minimum items, no need to continue
				break
			}
		}
	}

	// If no solution found
	if bestItems == -1 {
		return Result{PackCounts: make(map[int]int)}
	}

	// Backtrack to find which packs were used
	packCounts := make(map[int]int)
	current := bestItems
	for current > 0 {
		size := parent[current]
		packCounts[size]++
		current -= size
	}

	return Result{
		PackCounts: packCounts,
		TotalItems: bestItems,
		TotalPacks: bestPacks,
		Waste:      bestItems - order,
	}
}

// CalculateWithSizes is a convenience function that returns the result
// along with the pack sizes used
func CalculateWithSizes(order int, packSizes []int) (map[int]int, int, int, int) {
	result := Calculate(order, packSizes)
	return result.PackCounts, result.TotalItems, result.TotalPacks, result.Waste
}

// Validate checks if the given pack sizes can fulfill any order
func Validate(packSizes []int) bool {
	if len(packSizes) == 0 {
		return false
	}

	// Check if all pack sizes are positive
	for _, size := range packSizes {
		if size <= 0 {
			return false
		}
	}

	// Check if GCD of all pack sizes is 1
	// If GCD > 1, some orders cannot be fulfilled
	// For interview purposes, we'll allow any positive pack sizes
	return true
}

// gcd calculates the greatest common divisor
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// CalculateGCD finds the GCD of all pack sizes
func CalculateGCD(packSizes []int) int {
	if len(packSizes) == 0 {
		return 0
	}

	result := packSizes[0]
	for i := 1; i < len(packSizes); i++ {
		result = gcd(result, packSizes[i])
	}
	return result
}
