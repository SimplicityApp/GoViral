package service

import (
	"fmt"

	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/pkg/models"
)

// PersonaService provides read access to persona profiles.
type PersonaService struct {
	db *db.DB
}

// NewPersonaService creates a new PersonaService.
func NewPersonaService(database *db.DB) *PersonaService {
	return &PersonaService{db: database}
}

// Get returns the persona for the given platform.
func (s *PersonaService) Get(platform string) (*models.Persona, error) {
	persona, err := s.db.GetPersona(platform)
	if err != nil {
		return nil, fmt.Errorf("getting persona for %s: %w", platform, err)
	}
	return persona, nil
}
