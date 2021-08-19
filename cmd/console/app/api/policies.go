package api

import (
	"net/http"

	"github.com/common-fate/iamzero/api/io"
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/go-chi/chi"
)

// ListPolicies lists policies stored by IAM Zero.
// If the `status` query parameter is passed only policies matching the status
// will be returned
func (h *Handlers) ListPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")

	var policies []recommendations.Policy

	var err error
	if status != "" {
		if ok := recommendations.PolicyStatusIsValid(status); !ok {
			http.Error(w, "policy status must be 'active' or 'resolved'", http.StatusBadRequest)
			return
		}
		policies, err = h.PolicyStorage.ListForStatus(status)
	} else {
		policies, err = h.PolicyStorage.List()
	}
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	io.RespondJSON(ctx, h.Log, w, policies, http.StatusOK)
}

func (h *Handlers) GetPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	policyID := chi.URLParam(r, "policyID")
	policy, err := h.PolicyStorage.Get(policyID)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	if policy == nil {
		http.Error(w, "policy not found", http.StatusNotFound)
	} else {

		io.RespondJSON(ctx, h.Log, w, policy, http.StatusOK)
	}
}

func (h *Handlers) ListActionsForPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	policyID := chi.URLParam(r, "policyID")
	alerts, err := h.ActionStorage.ListForPolicy(policyID)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}
	io.RespondJSON(ctx, h.Log, w, alerts, http.StatusOK)
}

type setPolicyStatusBody struct {
	Status string `json:"status"`
}

// FindPolicy finds a policy by its role and status
func (h *Handlers) FindPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	role := r.URL.Query().Get("role")
	status := r.URL.Query().Get("status")

	if role == "" || status == "" {
		http.Error(w, "role and status must be provided as query parameters", http.StatusBadRequest)
		return
	}

	policy, err := h.PolicyStorage.FindByRole(storage.FindByRoleQuery{Role: role, Status: status})
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	if policy == nil {
		http.Error(w, "policy not found", http.StatusNotFound)
	} else {
		io.RespondJSON(ctx, h.Log, w, policy, http.StatusOK)
	}
}

func (h *Handlers) SetPolicyStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	policyID := chi.URLParam(r, "policyID")

	var b setPolicyStatusBody

	if err := io.DecodeJSONBody(w, r, &b); err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	if ok := recommendations.PolicyStatusIsValid(b.Status); !ok {
		http.Error(w, "policy status must be 'active' or 'resolved'", http.StatusBadRequest)
		return
	}

	policy, err := h.PolicyStorage.Get(policyID)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	if policy == nil {
		http.Error(w, "policy not found", http.StatusNotFound)
		return
	}

	policy.Status = b.Status

	err = h.PolicyStorage.CreateOrUpdate(*policy)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
	}

	io.RespondJSON(ctx, h.Log, w, policy, http.StatusOK)
}
