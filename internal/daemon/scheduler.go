package daemon

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// CronScheduler manages per-platform cron jobs.
type CronScheduler struct {
	c       *cron.Cron
	mu      sync.RWMutex
	entries map[string]cron.EntryID // platform → entry ID
}

// NewScheduler creates a new cron scheduler.
func NewScheduler() *CronScheduler {
	return &CronScheduler{
		c:       cron.New(),
		entries: make(map[string]cron.EntryID),
	}
}

// Add registers a cron job for a platform.
func (s *CronScheduler) Add(platform, cronExpr string, fn func()) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing entry for this platform if any
	if old, ok := s.entries[platform]; ok {
		s.c.Remove(old)
		delete(s.entries, platform)
	}

	id, err := s.c.AddFunc(cronExpr, fn)
	if err != nil {
		return fmt.Errorf("adding cron job for %s: %w", platform, err)
	}
	s.entries[platform] = id
	return nil
}

// Start begins the cron scheduler.
func (s *CronScheduler) Start(_ context.Context) {
	s.c.Start()
}

// Stop halts the cron scheduler and waits for running jobs.
func (s *CronScheduler) Stop() {
	ctx := s.c.Stop()
	<-ctx.Done()
}

// NextRun returns the next scheduled run time for a platform.
func (s *CronScheduler) NextRun(platform string) *time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.entries[platform]
	if !ok {
		return nil
	}
	entry := s.c.Entry(id)
	if entry.ID == 0 {
		return nil
	}
	t := entry.Next
	return &t
}
