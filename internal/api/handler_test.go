package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sander-remitly/pack-calc/internal/cache"
	"github.com/sander-remitly/pack-calc/internal/logger"
	"github.com/sander-remitly/pack-calc/internal/models"
	"github.com/sander-remitly/pack-calc/internal/repo"
)

func init() {
	// Initialize logger for tests
	logger.Initialize()
}

func setupTestHandler(t *testing.T) (*Handler, func()) {
	// Create temporary database
	dbPath := "test_packcalc_" + t.Name() + ".db"

	repository, err := repo.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Disable cache for most tests
	os.Setenv("REDIS_ENABLED", "false")
	cacheInstance := cache.NewCache()

	cleanup := func() {
		repository.Close()
		os.Remove(dbPath)
	}

	return NewHandler(repository, cacheInstance), cleanup
}

func TestHandleCalculate_ValidRequest(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	reqBody := models.CalculateRequest{
		Items:     250,
		PackSizes: []int{250, 500, 1000},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleCalculate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.CalculateResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Items != 250 {
		t.Errorf("Expected items 250, got %d", response.Items)
	}

	if response.TotalPacks != 1 {
		t.Errorf("Expected 1 pack, got %d", response.TotalPacks)
	}

	if response.Waste != 0 {
		t.Errorf("Expected 0 waste, got %d", response.Waste)
	}
}

func TestHandleCalculate_InvalidJSON(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleCalculate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCalculate_InvalidItems(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	tests := []struct {
		name  string
		items int
	}{
		{"Zero items", 0},
		{"Negative items", -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := models.CalculateRequest{
				Items:     tt.items,
				PackSizes: []int{250, 500},
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.HandleCalculate(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}
		})
	}
}

func TestHandleCalculate_DefaultPackSizes(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	reqBody := models.CalculateRequest{
		Items: 250,
		// No pack sizes - should use defaults
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleCalculate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.CalculateResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.PackSizes) == 0 {
		t.Error("Expected default pack sizes to be used")
	}
}

func TestHandlePresets(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/presets", nil)
	w := httptest.NewRecorder()

	handler.HandlePresets(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.PresetsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Presets) != 3 {
		t.Errorf("Expected 3 presets, got %d", len(response.Presets))
	}
}

func TestHandleHealth(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	handler.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response.Status)
	}

	if response.Database == "" {
		t.Error("Expected database status to be set")
	}
}

func TestHandleGetPackConfig(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/packs/config", nil)
	w := httptest.NewRecorder()

	handler.HandleGetPackConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.PackConfig
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.PackSizes) == 0 {
		t.Error("Expected pack sizes to be returned")
	}
}

func TestHandleUpdatePackConfig_Valid(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	reqBody := models.ConfigUpdateRequest{
		PackSizes: []int{100, 200, 300},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/packs/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleUpdatePackConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.ConfigUpdateResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.PackSizes) != 3 {
		t.Errorf("Expected 3 pack sizes, got %d", len(response.PackSizes))
	}
}

func TestHandleUpdatePackConfig_Invalid(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	tests := []struct {
		name      string
		packSizes []int
	}{
		{"Empty pack sizes", []int{}},
		{"Zero in pack sizes", []int{0, 250, 500}},
		{"Negative in pack sizes", []int{-100, 250, 500}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := models.ConfigUpdateRequest{
				PackSizes: tt.packSizes,
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/packs/config", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.HandleUpdatePackConfig(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}
		})
	}
}

func TestHandleCacheStats(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/cache/stats", nil)
	w := httptest.NewRecorder()

	handler.HandleCacheStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.CacheStatsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Cache should be disabled in tests
	if response.Enabled {
		t.Error("Expected cache to be disabled in tests")
	}
}

func TestHandleCacheClear(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/cache/clear", nil)
	w := httptest.NewRecorder()

	handler.HandleCacheClear(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCorsMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := corsMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected CORS header to be set")
	}
}
