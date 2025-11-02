package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sander-remitly/pack-calc/internal/logger"
	"go.uber.org/zap"
)

//go:embed templates/* static/*
var content embed.FS

// Handler handles web UI requests
type Handler struct {
	templates *template.Template
}

// NewHandler creates a new web handler
func NewHandler() (*Handler, error) {
	// Parse templates
	tmpl, err := template.ParseFS(content, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &Handler{
		templates: tmpl,
	}, nil
}

// SetupRoutes adds web UI routes to the router
func (h *Handler) SetupRoutes(r *chi.Mux) {
	// Serve static files
	staticFS, err := fs.Sub(content, "static")
	if err != nil {
		logger.Log.Fatal("Failed to create static filesystem", zap.Error(err))
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Serve index page
	r.Get("/", h.HandleIndex)
}

// HandleIndex serves the main UI page
func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		logger.Log.Error("Error rendering template", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
