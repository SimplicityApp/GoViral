package service

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/shuhao/goviral/apps/server/dto"
)

// Operation represents a long-running background operation.
type Operation struct {
	ID        string
	Status    string // "running", "completed", "failed"
	Result    interface{}
	Error     string
	CreatedAt time.Time
}

// OperationStore is an in-memory store for tracking long-running operations.
type OperationStore struct {
	mu    sync.RWMutex
	ops   map[string]*Operation
	ttl   time.Duration
}

// NewOperationStore creates a new OperationStore with the given TTL for completed operations.
func NewOperationStore(ttl time.Duration) *OperationStore {
	s := &OperationStore{
		ops: make(map[string]*Operation),
		ttl: ttl,
	}
	go s.cleanup()
	return s
}

// Create creates a new operation and returns its ID.
func (s *OperationStore) Create() string {
	id := generateOperationID()
	s.mu.Lock()
	s.ops[id] = &Operation{
		ID:        id,
		Status:    "running",
		CreatedAt: time.Now(),
	}
	s.mu.Unlock()
	return id
}

// Complete marks an operation as completed with a result.
func (s *OperationStore) Complete(id string, result interface{}) {
	s.mu.Lock()
	if op, ok := s.ops[id]; ok {
		op.Status = "completed"
		op.Result = result
	}
	s.mu.Unlock()
}

// Fail marks an operation as failed with an error message.
func (s *OperationStore) Fail(id string, errMsg string) {
	s.mu.Lock()
	if op, ok := s.ops[id]; ok {
		op.Status = "failed"
		op.Error = errMsg
	}
	s.mu.Unlock()
}

// Get returns the operation response for the given ID, or nil if not found.
func (s *OperationStore) Get(id string) *dto.OperationResponse {
	s.mu.RLock()
	op, ok := s.ops[id]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	return &dto.OperationResponse{
		ID:     op.ID,
		Status: op.Status,
		Result: op.Result,
		Error:  op.Error,
	}
}

func (s *OperationStore) cleanup() {
	ticker := time.NewTicker(s.ttl / 2)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-s.ttl)
		s.mu.Lock()
		for id, op := range s.ops {
			if op.Status != "running" && op.CreatedAt.Before(cutoff) {
				delete(s.ops, id)
			}
		}
		s.mu.Unlock()
	}
}

func generateOperationID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "op_" + hex.EncodeToString(b)
}
