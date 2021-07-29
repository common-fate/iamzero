package tokens

import (
	"context"

	"github.com/common-fate/iamzero/pkg/crypto"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// InMemoryTokenStorer is a token storage backend which stores tokens in memory.
// Should only be used for development and testing.
type InMemoryTokenStorer struct {
	log    *zap.SugaredLogger
	tracer trace.Tracer
	tokens []Token
}

// NewInMemoryTokenStorer initialises the in memory token storage
func NewInMemoryTokenStorer(ctx context.Context, log *zap.SugaredLogger, tracer trace.Tracer) *InMemoryTokenStorer {
	return &InMemoryTokenStorer{log, tracer, []Token{}}
}

// Create a Token and store it in memory
func (s *InMemoryTokenStorer) Create(ctx context.Context, name string) (*Token, error) {
	s.log.Info("creating token")

	ID, err := crypto.GenerateRandomToken()
	if err != nil {
		return nil, errors.Wrap(err, "generating token")
	}

	token := Token{
		ID:   ID,
		Name: name,
	}

	s.tokens = append(s.tokens, token)

	return &token, nil
}

// removes a token from the slice, preserving the order
func removeToken(slice []Token, i int) []Token {
	copy(slice[i:], slice[i+1:])
	return slice[:len(slice)-1]
}

// Delete a token
func (s *InMemoryTokenStorer) Delete(ctx context.Context, id string) error {
	s.log.Info("deleting token")

	for i, t := range s.tokens {
		if t.ID == id {
			s.tokens = removeToken(s.tokens, i)
			return nil
		}
	}
	return nil
}

// Get a token
func (s *InMemoryTokenStorer) Get(ctx context.Context, id string) (*Token, error) {

	for _, t := range s.tokens {
		if t.ID == id {
			return &t, nil
		}
	}

	return nil, nil
}

// List all tokens
func (s *InMemoryTokenStorer) List(ctx context.Context) ([]Token, error) {
	s.log.Info("listing tokens")
	_, span := s.tracer.Start(ctx, "InMemoryTokenStorer.List")
	defer span.End()

	return s.tokens, nil
}
