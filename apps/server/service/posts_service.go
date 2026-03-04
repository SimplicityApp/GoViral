package service

import (
	"fmt"

	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/pkg/models"
)

// PostsService provides read access to user posts.
type PostsService struct {
	db *db.DB
}

// NewPostsService creates a new PostsService.
func NewPostsService(database *db.DB) *PostsService {
	return &PostsService{db: database}
}

// List returns posts filtered by platform. If platform is empty, returns all.
// Limit of 0 means no limit.
func (s *PostsService) List(userID string, platform string, limit int) ([]models.Post, error) {
	var posts []models.Post
	var err error

	if platform != "" {
		posts, err = s.db.GetPostsByPlatform(userID, platform)
	} else {
		posts, err = s.db.GetAllPosts(userID)
	}
	if err != nil {
		return nil, fmt.Errorf("listing posts: %w", err)
	}

	if limit > 0 && len(posts) > limit {
		posts = posts[:limit]
	}
	return posts, nil
}
