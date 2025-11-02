package repo

import (
	"os"
	"testing"

	"github.com/sander-remitly/pack-calc/internal/logger"
)

func init() {
	// Initialize logger for tests
	logger.Initialize()
}

func setupTestRepo(t *testing.T) (*Repository, func()) {
	dbPath := "test_repo.db"

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	cleanup := func() {
		repo.Close()
		os.Remove(dbPath)
	}

	return repo, cleanup
}

func TestNew(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	if repo == nil {
		t.Fatal("Expected repository to be created")
	}

	if repo.db == nil {
		t.Fatal("Expected database connection to be established")
	}
}

func TestGetPackSizes_Default(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	sizes, err := repo.GetPackSizes()
	if err != nil {
		t.Fatalf("Failed to get pack sizes: %v", err)
	}

	// Should return default sizes when DB is empty
	if len(sizes) == 0 {
		t.Error("Expected default pack sizes to be returned")
	}

	expected := []int{250, 500, 1000, 2000, 5000}
	if len(sizes) != len(expected) {
		t.Errorf("Expected %d pack sizes, got %d", len(expected), len(sizes))
	}
}

func TestSetPackSizes(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	newSizes := []int{100, 200, 300, 400}
	err := repo.SetPackSizes(newSizes)
	if err != nil {
		t.Fatalf("Failed to set pack sizes: %v", err)
	}

	// Verify sizes were saved
	sizes, err := repo.GetPackSizes()
	if err != nil {
		t.Fatalf("Failed to get pack sizes: %v", err)
	}

	if len(sizes) != len(newSizes) {
		t.Errorf("Expected %d pack sizes, got %d", len(newSizes), len(sizes))
	}

	for i, size := range sizes {
		if size != newSizes[i] {
			t.Errorf("Expected pack size %d at index %d, got %d", newSizes[i], i, size)
		}
	}
}

func TestSetPackSizes_Replace(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	// Set initial sizes
	initialSizes := []int{100, 200, 300}
	err := repo.SetPackSizes(initialSizes)
	if err != nil {
		t.Fatalf("Failed to set initial pack sizes: %v", err)
	}

	// Replace with new sizes
	newSizes := []int{500, 1000}
	err = repo.SetPackSizes(newSizes)
	if err != nil {
		t.Fatalf("Failed to replace pack sizes: %v", err)
	}

	// Verify only new sizes exist
	sizes, err := repo.GetPackSizes()
	if err != nil {
		t.Fatalf("Failed to get pack sizes: %v", err)
	}

	if len(sizes) != len(newSizes) {
		t.Errorf("Expected %d pack sizes, got %d", len(newSizes), len(sizes))
	}
}

func TestSaveCalculation(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	items := 250
	packSizes := []int{250, 500, 1000}
	result := map[int]int{250: 1}
	totalItems := 250
	totalPacks := 1
	waste := 0

	err := repo.SaveCalculation(items, packSizes, result, totalItems, totalPacks, waste)
	if err != nil {
		t.Fatalf("Failed to save calculation: %v", err)
	}

	// Verify calculation was saved
	history, err := repo.GetHistory(10)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(history))
	}

	if history[0].Items != items {
		t.Errorf("Expected items %d, got %d", items, history[0].Items)
	}

	if history[0].TotalPacks != totalPacks {
		t.Errorf("Expected total packs %d, got %d", totalPacks, history[0].TotalPacks)
	}
}

func TestGetHistory(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	// Save multiple calculations
	for i := 1; i <= 5; i++ {
		err := repo.SaveCalculation(
			i*100,
			[]int{250, 500},
			map[int]int{250: i},
			i*100,
			i,
			0,
		)
		if err != nil {
			t.Fatalf("Failed to save calculation %d: %v", i, err)
		}
	}

	// Get history with limit
	history, err := repo.GetHistory(3)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("Expected 3 history entries, got %d", len(history))
	}

	// Verify we got 3 entries (order may vary if timestamps are identical)
	// Just check that we have valid data
	for _, entry := range history {
		if entry.Items <= 0 {
			t.Errorf("Expected positive items, got %d", entry.Items)
		}
		if entry.TotalPacks <= 0 {
			t.Errorf("Expected positive total packs, got %d", entry.TotalPacks)
		}
	}
}

func TestGetHistory_Empty(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	history, err := repo.GetHistory(10)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if len(history) != 0 {
		t.Errorf("Expected empty history, got %d entries", len(history))
	}
}

func TestClearHistory(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	// Save some calculations
	for i := 1; i <= 3; i++ {
		err := repo.SaveCalculation(
			i*100,
			[]int{250, 500},
			map[int]int{250: i},
			i*100,
			i,
			0,
		)
		if err != nil {
			t.Fatalf("Failed to save calculation %d: %v", i, err)
		}
	}

	// Clear history
	err := repo.ClearHistory()
	if err != nil {
		t.Fatalf("Failed to clear history: %v", err)
	}

	// Verify history is empty
	history, err := repo.GetHistory(10)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if len(history) != 0 {
		t.Errorf("Expected empty history after clear, got %d entries", len(history))
	}
}

func TestPing(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	err := repo.Ping()
	if err != nil {
		t.Errorf("Expected ping to succeed, got error: %v", err)
	}
}

func TestPing_ClosedConnection(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	cleanup() // Close connection immediately

	err := repo.Ping()
	if err == nil {
		t.Error("Expected ping to fail on closed connection")
	}
}

func TestSaveCalculation_ComplexResult(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	items := 12001
	packSizes := []int{250, 500, 1000, 2000, 5000}
	result := map[int]int{
		5000: 2,
		2000: 1,
		250:  1,
	}
	totalItems := 12250
	totalPacks := 4
	waste := 249

	err := repo.SaveCalculation(items, packSizes, result, totalItems, totalPacks, waste)
	if err != nil {
		t.Fatalf("Failed to save complex calculation: %v", err)
	}

	// Verify calculation was saved correctly
	history, err := repo.GetHistory(1)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("Expected 1 history entry, got %d", len(history))
	}

	entry := history[0]
	if entry.Items != items {
		t.Errorf("Expected items %d, got %d", items, entry.Items)
	}

	if entry.TotalPacks != totalPacks {
		t.Errorf("Expected total packs %d, got %d", totalPacks, entry.TotalPacks)
	}

	if entry.Waste != waste {
		t.Errorf("Expected waste %d, got %d", waste, entry.Waste)
	}

	// Verify result map
	if len(entry.Result) != len(result) {
		t.Errorf("Expected %d result entries, got %d", len(result), len(entry.Result))
	}

	for size, count := range result {
		if entry.Result[size] != count {
			t.Errorf("Expected result[%d] = %d, got %d", size, count, entry.Result[size])
		}
	}
}
