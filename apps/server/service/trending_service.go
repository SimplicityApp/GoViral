package service

import (
	"fmt"

	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/pkg/models"
)

// TrendingService provides read access to trending posts.
type TrendingService struct {
	db *db.DB
}

// NewTrendingService creates a new TrendingService.
func NewTrendingService(database *db.DB) *TrendingService {
	return &TrendingService{db: database}
}

// List returns trending posts filtered by platform with an optional limit.
func (s *TrendingService) List(platform string, limit int) ([]models.TrendingPost, error) {
	posts, err := s.db.GetTrendingPosts(platform, limit)
	if err != nil {
		return nil, fmt.Errorf("listing trending posts: %w", err)
	}
	return posts, nil
}

// GetByID returns a single trending post by ID.
func (s *TrendingService) GetByID(id int64) (*models.TrendingPost, error) {
	post, err := s.db.GetTrendingPostByID(id)
	if err != nil {
		return nil, fmt.Errorf("getting trending post %d: %w", id, err)
	}
	return post, nil
}
