package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestGetDefaultPackSizes(t *testing.T) {
	sizes := GetDefaultPackSizes()
	
	expected := []int{250, 500, 1000, 2000, 5000}
	if len(sizes) != len(expected) {
		t.Errorf("Expected %d pack sizes, got %d", len(expected), len(sizes))
	}
	
	for i, size := range sizes {
		if size != expected[i] {
			t.Errorf("Expected pack size %d at index %d, got %d", expected[i], i, size)
		}
	}
}

func TestGetPresets(t *testing.T) {
	presets := GetPresets()
	
	if len(presets) != 3 {
		t.Errorf("Expected 3 presets, got %d", len(presets))
	}
	
	// Test Standard preset
	if presets[0].Name != "Standard" {
		t.Errorf("Expected first preset name to be 'Standard', got '%s'", presets[0].Name)
	}
	
	expectedStandard := []int{250, 500, 1000, 2000, 5000}
	if len(presets[0].PackSizes) != len(expectedStandard) {
		t.Errorf("Expected Standard preset to have %d sizes, got %d", len(expectedStandard), len(presets[0].PackSizes))
	}
	
	// Test Edge Case preset
	if presets[1].Name != "Edge Case" {
		t.Errorf("Expected second preset name to be 'Edge Case', got '%s'", presets[1].Name)
	}
	
	// Test Small Packs preset
	if presets[2].Name != "Small Packs" {
		t.Errorf("Expected third preset name to be 'Small Packs', got '%s'", presets[2].Name)
	}
}

func TestCalculateRequestJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected CalculateRequest
		wantErr  bool
	}{
		{
			name:     "Valid request with pack sizes",
			json:     `{"items": 250, "pack_sizes": [250, 500, 1000]}`,
			expected: CalculateRequest{Items: 250, PackSizes: []int{250, 500, 1000}},
			wantErr:  false,
		},
		{
			name:     "Valid request without pack sizes",
			json:     `{"items": 100}`,
			expected: CalculateRequest{Items: 100, PackSizes: nil},
			wantErr:  false,
		},
		{
			name:     "Invalid JSON",
			json:     `{"items": "not a number"}`,
			expected: CalculateRequest{},
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req CalculateRequest
			err := json.Unmarshal([]byte(tt.json), &req)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if req.Items != tt.expected.Items {
					t.Errorf("Expected items %d, got %d", tt.expected.Items, req.Items)
				}
				
				if len(req.PackSizes) != len(tt.expected.PackSizes) {
					t.Errorf("Expected %d pack sizes, got %d", len(tt.expected.PackSizes), len(req.PackSizes))
				}
			}
		})
	}
}

func TestCalculateResponseJSON(t *testing.T) {
	response := CalculateResponse{
		Items:             250,
		PackSizes:         []int{250, 500, 1000},
		Result:            map[int]int{250: 1},
		TotalItems:        250,
		TotalPacks:        1,
		Waste:             0,
		CalculationTimeMs: 5,
		Cached:            false,
	}
	
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}
	
	var decoded CalculateResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	
	if decoded.Items != response.Items {
		t.Errorf("Expected items %d, got %d", response.Items, decoded.Items)
	}
	
	if decoded.TotalPacks != response.TotalPacks {
		t.Errorf("Expected total packs %d, got %d", response.TotalPacks, decoded.TotalPacks)
	}
	
	if decoded.Cached != response.Cached {
		t.Errorf("Expected cached %v, got %v", response.Cached, decoded.Cached)
	}
}

func TestHealthResponseJSON(t *testing.T) {
	now := time.Now()
	response := HealthResponse{
		Status:    "ok",
		Timestamp: now,
		Database:  "connected",
		Uptime:    "1h30m",
	}
	
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal health response: %v", err)
	}
	
	var decoded HealthResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal health response: %v", err)
	}
	
	if decoded.Status != response.Status {
		t.Errorf("Expected status %s, got %s", response.Status, decoded.Status)
	}
	
	if decoded.Database != response.Database {
		t.Errorf("Expected database %s, got %s", response.Database, decoded.Database)
	}
}

func TestErrorResponseJSON(t *testing.T) {
	response := ErrorResponse{
		Error:   "validation_error",
		Message: "Items must be greater than 0",
		Code:    400,
	}
	
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal error response: %v", err)
	}
	
	var decoded ErrorResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}
	
	if decoded.Error != response.Error {
		t.Errorf("Expected error %s, got %s", response.Error, decoded.Error)
	}
	
	if decoded.Code != response.Code {
		t.Errorf("Expected code %d, got %d", response.Code, decoded.Code)
	}
}

func TestConfigUpdateRequestJSON(t *testing.T) {
	request := ConfigUpdateRequest{
		PackSizes: []int{100, 200, 300},
	}
	
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal config update request: %v", err)
	}
	
	var decoded ConfigUpdateRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal config update request: %v", err)
	}
	
	if len(decoded.PackSizes) != len(request.PackSizes) {
		t.Errorf("Expected %d pack sizes, got %d", len(request.PackSizes), len(decoded.PackSizes))
	}
	
	for i, size := range decoded.PackSizes {
		if size != request.PackSizes[i] {
			t.Errorf("Expected pack size %d at index %d, got %d", request.PackSizes[i], i, size)
		}
	}
}

func TestHistoryEntryJSON(t *testing.T) {
	now := time.Now()
	entry := HistoryEntry{
		ID:         1,
		Items:      250,
		PackSizes:  []int{250, 500},
		Result:     map[int]int{250: 1},
		TotalItems: 250,
		TotalPacks: 1,
		Waste:      0,
		Timestamp:  now,
	}
	
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal history entry: %v", err)
	}
	
	var decoded HistoryEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal history entry: %v", err)
	}
	
	if decoded.ID != entry.ID {
		t.Errorf("Expected ID %d, got %d", entry.ID, decoded.ID)
	}
	
	if decoded.Items != entry.Items {
		t.Errorf("Expected items %d, got %d", entry.Items, decoded.Items)
	}
}

func TestCacheStatsResponseJSON(t *testing.T) {
	response := CacheStatsResponse{
		Enabled:    true,
		Hits:       100,
		Misses:     10,
		HitRate:    90.91,
		TotalKeys:  50,
		MemoryUsed: "1.5MB",
		Uptime:     "2h30m",
	}
	
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal cache stats response: %v", err)
	}
	
	var decoded CacheStatsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal cache stats response: %v", err)
	}
	
	if decoded.Enabled != response.Enabled {
		t.Errorf("Expected enabled %v, got %v", response.Enabled, decoded.Enabled)
	}
	
	if decoded.Hits != response.Hits {
		t.Errorf("Expected hits %d, got %d", response.Hits, decoded.Hits)
	}
	
	if decoded.HitRate != response.HitRate {
		t.Errorf("Expected hit rate %.2f, got %.2f", response.HitRate, decoded.HitRate)
	}
}

