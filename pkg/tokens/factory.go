package tokens

import (
	"context"
	"errors"
	"flag"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type TokensStoreFactory struct {
	TokenStorageBackend           string
	TokenStorageDynamoDBTableName string
}

type TokensFactorySetupOpts struct {
	Log    *zap.SugaredLogger
	Tracer trace.Tracer
}

func NewFactory() *TokensStoreFactory {
	return &TokensStoreFactory{}
}

// AddFlags configures CLI flags
func (f *TokensStoreFactory) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&f.TokenStorageBackend, "token-storage-backend", "dynamodb", "token storage backend (must be 'dynamodb' or 'inmemory')")
	fs.StringVar(&f.TokenStorageDynamoDBTableName, "token-storage-dynamodb-table-name", "dynamodb", "the token storage table name (only for DynamoDB token storage backend)")
}

func (f *TokensStoreFactory) GetTokensStore(ctx context.Context, opts *TokensFactorySetupOpts) (TokenStorer, error) {
	var tokenStore TokenStorer
	var err error

	if f.TokenStorageBackend != "dynamodb" && f.TokenStorageBackend != "inmemory" {
		return nil, errors.New("token storage type must by dynamodb or inmemory")
	}

	if f.TokenStorageBackend == "dynamodb" {
		tokenStore, err = NewDynamoDBTokenStorer(ctx, f.TokenStorageDynamoDBTableName, opts.Log, opts.Tracer)
		if err != nil {
			return nil, err
		}
	} else if f.TokenStorageBackend == "inmemory" {
		tokenStore = NewInMemoryTokenStorer(ctx, opts.Log, opts.Tracer)
	}
	return tokenStore, nil
}
