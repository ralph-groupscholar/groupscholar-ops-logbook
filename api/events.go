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

var (
	pool     *pgxpool.Pool
	poolErr  error
	poolOnce sync.Once
)

func Handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleList(w, r)
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

	input.Title = strings.TrimSpace(input.Title)
	input.Category = strings.TrimSpace(input.Category)
	input.Severity = strings.TrimSpace(input.Severity)
	input.Owner = strings.TrimSpace(input.Owner)
	input.Status = strings.TrimSpace(input.Status)
	input.Notes = strings.TrimSpace(input.Notes)

	if input.Title == "" || input.Category == "" || input.Severity == "" || input.Owner == "" || input.Status == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	occurredAt := time.Now().UTC()
	if input.Occurred != "" {
		parsed, err := time.Parse(time.RFC3339, input.Occurred)
		if err != nil {
			http.Error(w, "invalid occurred_at", http.StatusBadRequest)
			return
		}
		occurredAt = parsed
	}

	query := `
		insert into groupscholar_ops_logbook.events (occurred_at, title, category, severity, owner, status, notes)
		values ($1, $2, $3, $4, $5, $6, $7)
		returning id, occurred_at, title, category, severity, owner, status, notes, created_at
	`

	var e Event
	if err := db.QueryRow(ctx, query, occurredAt, input.Title, input.Category, input.Severity, input.Owner, input.Status, input.Notes).
		Scan(&e.ID, &e.Occurred, &e.Title, &e.Category, &e.Severity, &e.Owner, &e.Status, &e.Notes, &e.CreatedAt); err != nil {
		http.Error(w, "failed to create event", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, e)
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
