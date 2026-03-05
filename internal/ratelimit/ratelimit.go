package ratelimit

import (
	"errors"
	"fmt"

	"github.com/shuhao/goviral/internal/db"
)

// ErrRateLimited is returned when a user exceeds their daily AI request limit.
var ErrRateLimited = errors.New("daily AI request limit exceeded")

// CheckAIRateLimit checks whether the user has exceeded their daily AI usage cap.
// Only call this when the user is consuming the global (operator) key, not their own BYOK key.
func CheckAIRateLimit(database *db.DB, userID, provider string, dailyCap int) error {
	if dailyCap <= 0 {
		return nil // no limit configured
	}
	count, err := database.GetAIUsage(userID, provider)
	if err != nil {
		return fmt.Errorf("checking AI rate limit: %w", err)
	}
	if count >= dailyCap {
		return fmt.Errorf("%w: %d/%d requests used today for %s", ErrRateLimited, count, dailyCap, provider)
	}
	return nil
}

// RecordAIUsage increments the daily usage counter and returns the new count.
func RecordAIUsage(database *db.DB, userID, provider string) (int, error) {
	count, err := database.IncrementAIUsage(userID, provider)
	if err != nil {
		return 0, fmt.Errorf("recording AI usage: %w", err)
	}
	return count, nil
}
