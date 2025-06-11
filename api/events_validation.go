package handler

import (
	"strings"
	"time"
)

func normalizeInput(input EventInput) (EventInput, time.Time, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Category = strings.TrimSpace(input.Category)
	input.Severity = strings.TrimSpace(input.Severity)
	input.Owner = strings.TrimSpace(input.Owner)
	input.Status = strings.TrimSpace(input.Status)
	input.Notes = strings.TrimSpace(input.Notes)

	if input.Title == "" || input.Category == "" || input.Severity == "" || input.Owner == "" || input.Status == "" {
		return input, time.Time{}, errMissingFields
	}

	occurredAt := time.Now().UTC()
	if input.Occurred != "" {
		parsed, err := time.Parse(time.RFC3339, input.Occurred)
		if err != nil {
			return input, time.Time{}, errInvalidOccurredAt
		}
		occurredAt = parsed
	}

	return input, occurredAt, nil
}
