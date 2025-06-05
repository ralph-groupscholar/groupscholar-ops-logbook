package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("DATABASE_URL is required")
		os.Exit(1)
	}

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		fmt.Printf("connect failed: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	schemaSQL, err := os.ReadFile("db/schema.sql")
	if err != nil {
		fmt.Printf("read schema failed: %v\n", err)
		os.Exit(1)
	}

	if _, err := conn.Exec(ctx, string(schemaSQL)); err != nil {
		fmt.Printf("apply schema failed: %v\n", err)
		os.Exit(1)
	}

	if _, err := conn.Exec(ctx, "delete from groupscholar_ops_logbook.events"); err != nil {
		fmt.Printf("clear data failed: %v\n", err)
		os.Exit(1)
	}

	seed := `
		insert into groupscholar_ops_logbook.events
		(occurred_at, title, category, severity, owner, status, notes)
		values
		(now() - interval '2 days', 'Mentor intro pipeline delay', 'Engagement', 'Medium', 'Mentor Ops', 'Monitoring', 'Vendor sync lag created a 36-hour delay.'),
		(now() - interval '30 hours', 'Scholar portal outage', 'Systems', 'High', 'Platform', 'Resolved', 'Restarted queue workers and cleared cache.'),
		(now() - interval '22 hours', 'FAFSA reminder bounce spike', 'Comms', 'Low', 'Comms Team', 'Open', 'Segmented list with higher bounce rate; validating emails.'),
		(now() - interval '16 hours', 'Cohort kickoff attendance dip', 'Programs', 'Medium', 'Program Lead', 'Monitoring', 'Sent follow-up RSVP and rescheduled a Q&A.'),
		(now() - interval '9 hours', 'Partner data feed mismatch', 'Data', 'High', 'Data Ops', 'Open', 'Records mismatch between partner feed and intake export.'),
		(now() - interval '3 hours', 'Scholar escalation resolved', 'Support', 'Low', 'Support Team', 'Resolved', 'Escalation closed after grant timeline clarification.')
	`

	if _, err := conn.Exec(ctx, seed); err != nil {
		fmt.Printf("seed data failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Seed complete")
}
