package repo

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sander-remitly/pack-calc/internal/logger"
	"github.com/sander-remitly/pack-calc/internal/models"
	"go.uber.org/zap"
)

// Repository handles data persistence
type Repository struct {
	db *sql.DB
}

// New creates a new repository instance
func New(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &Repository{db: db}
	if err := repo.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return repo, nil
}

// initialize creates the database schema
func (r *Repository) initialize() error {
	schema := `
	CREATE TABLE IF NOT EXISTS pack_sizes (
		size INTEGER PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS calculations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		items INTEGER NOT NULL,
		pack_sizes TEXT NOT NULL,
		result TEXT NOT NULL,
		total_items INTEGER NOT NULL,
		total_packs INTEGER NOT NULL,
		waste INTEGER NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_calculations_timestamp ON calculations(timestamp DESC);
	`

	_, err := r.db.Exec(schema)
	return err
}

// Close closes the database connection
func (r *Repository) Close() error {
	return r.db.Close()
}

// GetPackSizes retrieves the current pack sizes from the database
func (r *Repository) GetPackSizes() ([]int, error) {
	rows, err := r.db.Query("SELECT size FROM pack_sizes ORDER BY size")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sizes []int
	for rows.Next() {
		var size int
		if err := rows.Scan(&size); err != nil {
			return nil, err
		}
		sizes = append(sizes, size)
	}

	// If no pack sizes in DB, return default
	if len(sizes) == 0 {
		return models.GetDefaultPackSizes(), nil
	}

	return sizes, nil
}

// SetPackSizes updates the pack sizes in the database
func (r *Repository) SetPackSizes(sizes []int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing pack sizes
	if _, err := tx.Exec("DELETE FROM pack_sizes"); err != nil {
		return err
	}

	// Insert new pack sizes
	stmt, err := tx.Prepare("INSERT INTO pack_sizes (size) VALUES (?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, size := range sizes {
		if _, err := stmt.Exec(size); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SaveCalculation saves a calculation to the history
func (r *Repository) SaveCalculation(
	items int,
	packSizes []int,
	result map[int]int,
	totalItems, totalPacks, waste int,
) error {
	packSizesJSON, err := json.Marshal(packSizes)
	if err != nil {
		return err
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO calculations (items, pack_sizes, result, total_items, total_packs, waste)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.Exec(query, items, packSizesJSON, resultJSON, totalItems, totalPacks, waste)
	return err
}

// GetHistory retrieves the calculation history
func (r *Repository) GetHistory(limit int) ([]models.HistoryEntry, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, items, pack_sizes, result, total_items, total_packs, waste, timestamp
		FROM calculations
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.HistoryEntry
	for rows.Next() {
		var entry models.HistoryEntry
		var packSizesJSON, resultJSON string

		err := rows.Scan(
			&entry.ID,
			&entry.Items,
			&packSizesJSON,
			&resultJSON,
			&entry.TotalItems,
			&entry.TotalPacks,
			&entry.Waste,
			&entry.Timestamp,
		)
		if err != nil {
			logger.Log.Warn("Error scanning row", zap.Error(err))
			continue
		}

		if err := json.Unmarshal([]byte(packSizesJSON), &entry.PackSizes); err != nil {
			logger.Log.Warn("Error unmarshaling pack sizes", zap.Error(err))
			continue
		}

		if err := json.Unmarshal([]byte(resultJSON), &entry.Result); err != nil {
			logger.Log.Warn("Error unmarshaling result", zap.Error(err))
			continue
		}

		history = append(history, entry)
	}

	return history, nil
}

// ClearHistory clears all calculation history
func (r *Repository) ClearHistory() error {
	_, err := r.db.Exec("DELETE FROM calculations")
	return err
}

// GetStats returns statistics about the database
func (r *Repository) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count total calculations
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM calculations").Scan(&count)
	if err != nil {
		return nil, err
	}
	stats["total_calculations"] = count

	// Get pack sizes count
	var packSizesCount int
	err = r.db.QueryRow("SELECT COUNT(*) FROM pack_sizes").Scan(&packSizesCount)
	if err != nil {
		return nil, err
	}
	stats["pack_sizes_count"] = packSizesCount

	// Get latest calculation time
	var latestTime time.Time
	err = r.db.QueryRow("SELECT MAX(timestamp) FROM calculations").Scan(&latestTime)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if !latestTime.IsZero() {
		stats["latest_calculation"] = latestTime
	}

	return stats, nil
}

// Ping checks if the database connection is alive
func (r *Repository) Ping() error {
	return r.db.Ping()
}
