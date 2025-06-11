package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ID        int       `json:"id"`
	Occurred  time.Time `json:"occurred_at"`
	Title     string    `json:"title"`
	Category  string    `json:"category"`
	Severity  string    `json:"severity"`
	Owner     string    `json:"owner"`
	Status    string    `json:"status"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

type EventInput struct {
	Occurred string `json:"occurred_at"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Severity string `json:"severity"`
	Owner    string `json:"owner"`
	Status   string `json:"status"`
	Notes    string `json:"notes"`
}

type Summary struct {
	TotalCount       int        `json:"total_count"`
	OpenCount        int        `json:"open_count"`
	MonitoringCount  int        `json:"monitoring_count"`
	ResolvedCount    int        `json:"resolved_count"`
	HighCount        int        `json:"high_count"`
	MediumCount      int        `json:"medium_count"`
	LowCount         int        `json:"low_count"`
	TopCategory      string     `json:"top_category"`
	TopCategoryCount int        `json:"top_category_count"`
	TopOwner         string     `json:"top_owner"`
	TopOwnerCount    int        `json:"top_owner_count"`
	LatestOccurred   *time.Time `json:"latest_occurred,omitempty"`
	AppliedStatus    string     `json:"applied_status"`
	AppliedCategory  string     `json:"applied_category"`
}

var (
	pool     *pgxpool.Pool
	poolErr  error
	poolOnce sync.Once

	errMissingFields     = errors.New("missing required fields")
	errInvalidOccurredAt = errors.New("invalid occurred_at")
)

func Handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("view") == "summary" {
			handleSummary(w, r)
		} else {
			handleList(w, r)
		}
	case http.MethodPost:
		handleCreate(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()

	db, err := getPool(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	category := strings.TrimSpace(r.URL.Query().Get("category"))

	query := `
		select id, occurred_at, title, category, severity, owner, status, notes, created_at
		from groupscholar_ops_logbook.events
		where ($1 = '' or status = $1)
		and ($2 = '' or category = $2)
		order by occurred_at desc
		limit 200
	`

	rows, err := db.Query(ctx, query, status, category)
	if err != nil {
		http.Error(w, "failed to load events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	events := make([]Event, 0, 32)
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Occurred, &e.Title, &e.Category, &e.Severity, &e.Owner, &e.Status, &e.Notes, &e.CreatedAt); err != nil {
			http.Error(w, "failed to parse events", http.StatusInternalServerError)
			return
		}
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "failed to read events", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, events)
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()

	db, err := getPool(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var input EventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	normalized, occurredAt, err := normalizeInput(input)
	if err != nil {
		if errors.Is(err, errMissingFields) {
			http.Error(w, "missing required fields", http.StatusBadRequest)
			return
		}
		if errors.Is(err, errInvalidOccurredAt) {
			http.Error(w, "invalid occurred_at", http.StatusBadRequest)
			return
		}
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	query := `
		insert into groupscholar_ops_logbook.events (occurred_at, title, category, severity, owner, status, notes)
		values ($1, $2, $3, $4, $5, $6, $7)
		returning id, occurred_at, title, category, severity, owner, status, notes, created_at
	`

	var e Event
	if err := db.QueryRow(ctx, query, occurredAt, normalized.Title, normalized.Category, normalized.Severity, normalized.Owner, normalized.Status, normalized.Notes).
		Scan(&e.ID, &e.Occurred, &e.Title, &e.Category, &e.Severity, &e.Owner, &e.Status, &e.Notes, &e.CreatedAt); err != nil {
		http.Error(w, "failed to create event", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, e)
}

func handleSummary(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()

	db, err := getPool(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	category := strings.TrimSpace(r.URL.Query().Get("category"))

	summary := Summary{
		AppliedStatus:   status,
		AppliedCategory: category,
	}

	summaryQuery := `
		select
			count(*) as total_count,
			count(*) filter (where status = 'Open') as open_count,
			count(*) filter (where status = 'Monitoring') as monitoring_count,
			count(*) filter (where status = 'Resolved') as resolved_count,
			count(*) filter (where severity = 'High') as high_count,
			count(*) filter (where severity = 'Medium') as medium_count,
			count(*) filter (where severity = 'Low') as low_count,
			max(occurred_at) as latest_occurred
		from groupscholar_ops_logbook.events
		where ($1 = '' or status = $1)
		and ($2 = '' or category = $2)
	`

	var latest pgtype.Timestamptz
	if err := db.QueryRow(ctx, summaryQuery, status, category).
		Scan(&summary.TotalCount, &summary.OpenCount, &summary.MonitoringCount, &summary.ResolvedCount, &summary.HighCount, &summary.MediumCount, &summary.LowCount, &latest); err != nil {
		http.Error(w, "failed to load summary", http.StatusInternalServerError)
		return
	}

	if latest.Valid {
		t := latest.Time
		summary.LatestOccurred = &t
	}

	topCategoryQuery := `
		select category, count(*) as total
		from groupscholar_ops_logbook.events
		where ($1 = '' or status = $1)
		and ($2 = '' or category = $2)
		group by category
		order by total desc
		limit 1
	`

	if err := db.QueryRow(ctx, topCategoryQuery, status, category).
		Scan(&summary.TopCategory, &summary.TopCategoryCount); err != nil && err != pgx.ErrNoRows {
		http.Error(w, "failed to load summary", http.StatusInternalServerError)
		return
	}

	topOwnerQuery := `
		select owner, count(*) as total
		from groupscholar_ops_logbook.events
		where ($1 = '' or status = $1)
		and ($2 = '' or category = $2)
		group by owner
		order by total desc
		limit 1
	`

	if err := db.QueryRow(ctx, topOwnerQuery, status, category).
		Scan(&summary.TopOwner, &summary.TopOwnerCount); err != nil && err != pgx.ErrNoRows {
		http.Error(w, "failed to load summary", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func getPool(ctx context.Context) (*pgxpool.Pool, error) {
	poolOnce.Do(func() {
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			poolErr = errors.New("DATABASE_URL not set")
			return
		}

		cfg, err := pgxpool.ParseConfig(dbURL)
		if err != nil {
			poolErr = err
			return
		}
		cfg.MaxConns = 4
		cfg.MinConns = 0
		cfg.MaxConnLifetime = 5 * time.Minute
		cfg.MaxConnIdleTime = 1 * time.Minute

		pool, poolErr = pgxpool.NewWithConfig(ctx, cfg)
	})

	if poolErr != nil {
		return nil, poolErr
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return pool, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
