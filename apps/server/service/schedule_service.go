package service

import (
	"fmt"

	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/pkg/models"
)

// ScheduleService provides read access to scheduled posts.
type ScheduleService struct {
	db *db.DB
}

// NewScheduleService creates a new ScheduleService.
func NewScheduleService(database *db.DB) *ScheduleService {
	return &ScheduleService{db: database}
}

// List returns scheduled posts with optional status filter and limit.
func (s *ScheduleService) List(status string, limit int) ([]models.ScheduledPost, error) {
	posts, err := s.db.GetScheduledPosts(status, limit)
	if err != nil {
		return nil, fmt.Errorf("listing scheduled posts: %w", err)
	}
	return posts, nil
}
