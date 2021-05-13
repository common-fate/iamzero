package api

import (
	"context"
	"net/http"

	"go.uber.org/zap"
	"iamzero.dev/server/api/io"
)

// Check provides support for orchestration health checks.
type Check struct {
	Log *zap.SugaredLogger
}

// Health validates the service is healthy and ready to accept requests.
func (c *Check) Health(w http.ResponseWriter, r *http.Request) {
	health := struct {
		Status string `json:"status"`
	}{}
	ctx := context.TODO()

	health.Status = "ok"
	io.RespondJSON(ctx, c.Log, w, health, http.StatusOK)
}
