package tokens

import (
	"context"

	"github.com/pkg/errors"
)

// Token is a token which allows IAM Zero clients to send events to IAM Zero
type Token struct {
	ID   string `dynamodbav:"id" json:"id"`
	Name string `dynamodbav:"name" json:"name"`
}

var ErrTokenNotFound = errors.New("token not found")

// TokenStorer stores and loads Tokens
type TokenStorer interface {
	Create(ctx context.Context, name string) (*Token, error)
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*Token, error)
	List(ctx context.Context) ([]Token, error)
}
