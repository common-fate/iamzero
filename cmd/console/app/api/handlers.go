package api

import (
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/common-fate/iamzero/pkg/tokens"
	"go.uber.org/zap"
)

// Handlers holds all of the REST API endpoints for the console
type Handlers struct {
	Log           *zap.SugaredLogger
	TokenStore    tokens.TokenStorer
	ActionStorage *storage.ActionStorage
	PolicyStorage *storage.PolicyStorage
}
