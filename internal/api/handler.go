package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sander-remitly/pack-calc/internal/algorithm"
	"github.com/sander-remitly/pack-calc/internal/cache"
	"github.com/sander-remitly/pack-calc/internal/logger"
	"github.com/sander-remitly/pack-calc/internal/models"
	"github.com/sander-remitly/pack-calc/internal/repo"
	"go.uber.org/zap"
)

// Handler handles HTTP requests
type Handler struct {
	repo      *repo.Repository
	cache     *cache.Cache
	startTime time.Time
}

// NewHandler creates a new API handler
func NewHandler(repository *repo.Repository, cacheInstance *cache.Cache) *Handler {
	return &Handler{
		repo:      repository,
		cache:     cacheInstance,
		startTime: time.Now(),
	}
}

// SetupRouter configures the Chi router with all routes
func (h *Handler) SetupRouter() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(corsMiddleware)

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Post("/calculate", h.HandleCalculate)
		r.Get("/presets", h.HandlePresets)
		r.Get("/history", h.HandleHistory)
		r.Post("/history/clear", h.HandleClearHistory)
		r.Get("/health", h.HandleHealth)
		r.Get("/packs/config", h.HandleGetPackConfig)
		r.Post("/packs/config", h.HandleUpdatePackConfig)

		// Cache endpoints
		r.Get("/cache/stats", h.HandleCacheStats)
		r.Post("/cache/clear", h.HandleCacheClear)
	})

	return r
}

// HandleCalculate handles pack calculation requests
func (h *Handler) HandleCalculate(w http.ResponseWriter, r *http.Request) {
	var req models.CalculateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate input
	if req.Items <= 0 {
		respondError(w, http.StatusBadRequest, "Items must be greater than 0", nil)
		return
	}

	// Get pack sizes (use provided or default from DB)
	packSizes := req.PackSizes
	if len(packSizes) == 0 {
		var err error
		packSizes, err = h.repo.GetPackSizes()
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to get pack sizes", err)
			return
		}
	}

	// Validate pack sizes
	if !algorithm.Validate(packSizes) {
		respondError(w, http.StatusBadRequest, "Invalid pack sizes", nil)
		return
	}

	// Try to get from cache first
	if cached, found := h.cache.Get(req.Items, packSizes); found {
		logger.Log.Info("Cache HIT",
			zap.Int("items", req.Items),
			zap.Ints("pack_sizes", packSizes),
			zap.Int("hit_count", cached.HitCount),
			zap.Duration("ttl", cached.CurrentTTL),
		)

		response := models.CalculateResponse{
			Items:             cached.Items,
			PackSizes:         cached.PackSizes,
			Result:            cached.Result,
			TotalItems:        cached.TotalItems,
			TotalPacks:        cached.TotalPacks,
			Waste:             cached.Waste,
			CalculationTimeMs: cached.CalculationTimeMs,
			Cached:            true,
			CacheTTL:          cached.CurrentTTL.String(),
			CacheHitCount:     cached.HitCount,
		}

		respondJSON(w, http.StatusOK, response)
		return
	}

	logger.Log.Info("Cache MISS",
		zap.Int("items", req.Items),
		zap.Ints("pack_sizes", packSizes),
	)

	// Calculate
	start := time.Now()
	result := algorithm.Calculate(req.Items, packSizes)
	duration := time.Since(start)

	// Save to cache
	if err := h.cache.Set(
		req.Items,
		packSizes,
		result.PackCounts,
		result.TotalItems,
		result.TotalPacks,
		result.Waste,
		duration.Milliseconds(),
	); err != nil {
		logger.Log.Warn("Failed to cache result", zap.Error(err))
	}

	// Save to history
	if err := h.repo.SaveCalculation(
		req.Items,
		packSizes,
		result.PackCounts,
		result.TotalItems,
		result.TotalPacks,
		result.Waste,
	); err != nil {
		logger.Log.Warn("Failed to save calculation", zap.Error(err))
		// Don't fail the request, just log
	}

	// Build response
	response := models.CalculateResponse{
		Items:             req.Items,
		PackSizes:         packSizes,
		Result:            result.PackCounts,
		TotalItems:        result.TotalItems,
		TotalPacks:        result.TotalPacks,
		Waste:             result.Waste,
		CalculationTimeMs: duration.Milliseconds(),
		Cached:            false,
	}

	respondJSON(w, http.StatusOK, response)
}

// HandlePresets returns predefined pack size configurations
func (h *Handler) HandlePresets(w http.ResponseWriter, r *http.Request) {
	response := models.PresetsResponse{
		Presets: models.GetPresets(),
	}
	respondJSON(w, http.StatusOK, response)
}

// HandleHistory returns calculation history
func (h *Handler) HandleHistory(w http.ResponseWriter, r *http.Request) {
	history, err := h.repo.GetHistory(20)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get history", err)
		return
	}

	response := models.HistoryResponse{
		History: history,
		Count:   len(history),
	}
	respondJSON(w, http.StatusOK, response)
}

// HandleClearHistory clears all calculation history
func (h *Handler) HandleClearHistory(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.ClearHistory(); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to clear history", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "History cleared"})
}

// HandleHealth returns service health status
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	dbStatus := "connected"
	if err := h.repo.Ping(); err != nil {
		dbStatus = "disconnected"
	}

	uptime := time.Since(h.startTime).Round(time.Second).String()

	response := models.HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Database:  dbStatus,
		Uptime:    uptime,
	}

	respondJSON(w, http.StatusOK, response)
}

// HandleGetPackConfig returns the current pack configuration
func (h *Handler) HandleGetPackConfig(w http.ResponseWriter, r *http.Request) {
	packSizes, err := h.repo.GetPackSizes()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get pack config", err)
		return
	}

	response := models.PackConfig{
		PackSizes: packSizes,
		UpdatedAt: time.Now(),
	}

	respondJSON(w, http.StatusOK, response)
}

// HandleUpdatePackConfig updates the pack size configuration
func (h *Handler) HandleUpdatePackConfig(w http.ResponseWriter, r *http.Request) {
	var req models.ConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate pack sizes
	if !algorithm.Validate(req.PackSizes) {
		respondError(w, http.StatusBadRequest, "Invalid pack sizes", nil)
		return
	}

	// Update in database
	if err := h.repo.SetPackSizes(req.PackSizes); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update pack config", err)
		return
	}

	response := models.ConfigUpdateResponse{
		PackSizes: req.PackSizes,
		UpdatedAt: time.Now(),
		Message:   "Pack sizes updated successfully",
	}

	respondJSON(w, http.StatusOK, response)
}

// HandleCacheStats returns cache statistics
func (h *Handler) HandleCacheStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.cache.GetStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get cache stats", err)
		return
	}

	response := models.CacheStatsResponse{
		Enabled:    h.cache.IsEnabled(),
		Hits:       stats.Hits,
		Misses:     stats.Misses,
		HitRate:    stats.HitRate,
		TotalKeys:  stats.TotalKeys,
		MemoryUsed: stats.MemoryUsed,
		Uptime:     stats.Uptime,
	}

	respondJSON(w, http.StatusOK, response)
}

// HandleCacheClear clears all cache entries
func (h *Handler) HandleCacheClear(w http.ResponseWriter, r *http.Request) {
	if err := h.cache.Clear(); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to clear cache", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Cache cleared successfully"})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Log.Error("Error encoding JSON response", zap.Error(err))
	}
}

func respondError(w http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		logger.Log.Error("Request error",
			zap.String("message", message),
			zap.Int("status", status),
			zap.Error(err),
		)
	}

	response := models.ErrorResponse{
		Error: message,
		Code:  status,
	}

	if err != nil {
		response.Message = err.Error()
	}

	respondJSON(w, status, response)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
