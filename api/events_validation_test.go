package handler

import (
	"testing"
	"time"
)

func TestNormalizeInputMissingFields(t *testing.T) {
	_, _, err := normalizeInput(EventInput{
		Title: " ",
	})
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
	if err != errMissingFields {
		t.Fatalf("expected missing fields error, got %v", err)
	}
}

func TestNormalizeInputInvalidOccurredAt(t *testing.T) {
	_, _, err := normalizeInput(EventInput{
		Title:    "Escalation",
		Category: "Operations",
		Severity: "High",
		Owner:    "Ops",
		Status:   "Open",
		Occurred: "nope",
	})
	if err == nil {
		t.Fatal("expected error for invalid occurred_at")
	}
	if err != errInvalidOccurredAt {
		t.Fatalf("expected invalid occurred_at error, got %v", err)
	}
}

func TestNormalizeInputSuccess(t *testing.T) {
	input := EventInput{
		Title:    "  Escalation  ",
		Category: " Operations ",
		Severity: " High ",
		Owner:    " Ops ",
		Status:   " Open ",
		Notes:    "  Notes  ",
		Occurred: "2026-02-08T12:30:00Z",
	}
	normalized, occurredAt, err := normalizeInput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalized.Title != "Escalation" || normalized.Category != "Operations" || normalized.Severity != "High" || normalized.Owner != "Ops" || normalized.Status != "Open" || normalized.Notes != "Notes" {
		t.Fatalf("unexpected normalized values: %+v", normalized)
	}
	expected, _ := time.Parse(time.RFC3339, input.Occurred)
	if !occurredAt.Equal(expected) {
		t.Fatalf("expected occurredAt %v, got %v", expected, occurredAt)
	}
}
