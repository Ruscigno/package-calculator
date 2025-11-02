package models

import "time"

// CalculateRequest represents the API request for pack calculation
type CalculateRequest struct {
	Items     int   `json:"items"`
	PackSizes []int `json:"pack_sizes,omitempty"` // Optional: use default if not provided
}

// CalculateResponse represents the API response for pack calculation
type CalculateResponse struct {
	Items             int         `json:"items"`                     // Original order quantity
	PackSizes         []int       `json:"pack_sizes"`                // Pack sizes used
	Result            map[int]int `json:"result"`                    // Pack size -> count
	TotalItems        int         `json:"total_items"`               // Total items delivered
	TotalPacks        int         `json:"total_packs"`               // Total number of packs
	Waste             int         `json:"waste"`                     // Excess items
	CalculationTimeMs int64       `json:"calculation_time_ms"`       // Time taken in milliseconds
	Cached            bool        `json:"cached"`                    // Whether result was from cache
	CacheTTL          string      `json:"cache_ttl,omitempty"`       // Current cache TTL (if cached)
	CacheHitCount     int         `json:"cache_hit_count,omitempty"` // Number of times this result was cached
}

// PackConfig represents the pack size configuration
type PackConfig struct {
	PackSizes []int     `json:"pack_sizes"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Preset represents a predefined pack size configuration
type Preset struct {
	Name      string `json:"name"`
	PackSizes []int  `json:"pack_sizes"`
}

// PresetsResponse represents the API response for presets
type PresetsResponse struct {
	Presets []Preset `json:"presets"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Database  string    `json:"database,omitempty"`
	Uptime    string    `json:"uptime,omitempty"`
}

// HistoryEntry represents a calculation history entry
type HistoryEntry struct {
	ID         int         `json:"id"`
	Items      int         `json:"items"`
	PackSizes  []int       `json:"pack_sizes"`
	Result     map[int]int `json:"result"`
	TotalItems int         `json:"total_items"`
	TotalPacks int         `json:"total_packs"`
	Waste      int         `json:"waste"`
	Timestamp  time.Time   `json:"timestamp"`
}

// HistoryResponse represents the API response for history
type HistoryResponse struct {
	History []HistoryEntry `json:"history"`
	Count   int            `json:"count"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// ConfigUpdateRequest represents a request to update pack sizes
type ConfigUpdateRequest struct {
	PackSizes []int `json:"pack_sizes"`
}

// ConfigUpdateResponse represents the response after updating pack sizes
type ConfigUpdateResponse struct {
	PackSizes []int     `json:"pack_sizes"`
	UpdatedAt time.Time `json:"updated_at"`
	Message   string    `json:"message"`
}

// CacheStatsResponse represents cache statistics
type CacheStatsResponse struct {
	Enabled    bool    `json:"enabled"`
	Hits       int64   `json:"hits"`
	Misses     int64   `json:"misses"`
	HitRate    float64 `json:"hit_rate"`
	TotalKeys  int64   `json:"total_keys"`
	MemoryUsed string  `json:"memory_used"`
	Uptime     string  `json:"uptime"`
}

// GetDefaultPackSizes returns the standard pack sizes
func GetDefaultPackSizes() []int {
	return []int{250, 500, 1000, 2000, 5000}
}

// GetPresets returns predefined pack size configurations
func GetPresets() []Preset {
	return []Preset{
		{
			Name:      "Standard",
			PackSizes: []int{250, 500, 1000, 2000, 5000},
		},
		{
			Name:      "Edge Case",
			PackSizes: []int{23, 31, 53},
		},
		{
			Name:      "Small Packs",
			PackSizes: []int{10, 25, 50, 100},
		},
	}
}
