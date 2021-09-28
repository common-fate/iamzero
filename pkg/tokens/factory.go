package tokens

import (
	"context"
	"errors"
	"flag"

	"github.com/jmoiron/sqlx"
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
	DB     *sqlx.DB
}

func NewFactory() *TokensStoreFactory {
	return &TokensStoreFactory{}
}

// AddFlags configures CLI flags
func (f *TokensStoreFactory) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&f.TokenStorageBackend, "token-storage-backend", "dynamodb", "token storage backend (must be 'dynamodb', 'inmemory' or 'postgres')")
	fs.StringVar(&f.TokenStorageDynamoDBTableName, "token-storage-dynamodb-table-name", "dynamodb", "the token storage table name (only for DynamoDB token storage backend)")
}

func (f *TokensStoreFactory) GetTokensStore(ctx context.Context, opts *TokensFactorySetupOpts) (TokenStorer, error) {
	var tokenStore TokenStorer
	var err error

	if f.TokenStorageBackend != "dynamodb" && f.TokenStorageBackend != "inmemory" && f.TokenStorageBackend != "postgres" {
		return nil, errors.New("token storage type must be dynamodb, inmemory, or postgres")
	}

	if f.TokenStorageBackend == "dynamodb" {
		tokenStore, err = NewDynamoDBTokenStorer(ctx, f.TokenStorageDynamoDBTableName, opts.Log, opts.Tracer)
		if err != nil {
			return nil, err
		}
	} else if f.TokenStorageBackend == "inmemory" {
		tokenStore = NewInMemoryTokenStorer(ctx, opts.Log, opts.Tracer)
	} else if f.TokenStorageBackend == "postgres" {
		tokenStore, err = NewPostgresDBTokenStorer(ctx, opts.DB, opts.Log, opts.Tracer)
		if err != nil {
			return nil, err
		}
	}
	return tokenStore, nil
}
