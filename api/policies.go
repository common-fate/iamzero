package api

import (
	"net/http"

	"github.com/common-fate/iamzero/api/io"
	"github.com/go-chi/chi"
)

func (h *Handlers) ListPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	policies := h.PolicyStorage.List()
	io.RespondJSON(ctx, h.Log, w, policies, http.StatusOK)
}

func (h *Handlers) GetPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	policyID := chi.URLParam(r, "policyID")
	policy := h.PolicyStorage.Get(policyID)
	io.RespondJSON(ctx, h.Log, w, policy, http.StatusOK)
}

func (h *Handlers) ListActionsForPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	policyID := chi.URLParam(r, "policyID")
	alerts := h.ActionStorage.ListForPolicy(policyID)
	io.RespondJSON(ctx, h.Log, w, alerts, http.StatusOK)
}
