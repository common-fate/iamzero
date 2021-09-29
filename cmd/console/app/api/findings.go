package api

import (
	"net/http"

	"github.com/common-fate/iamzero/api/io"
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/common-fate/iamzero/pkg/storage"
	"github.com/go-chi/chi"
)

// ListFindings lists findings stored by IAM Zero.
// If the `status` query parameter is passed only findings matching the status
// will be returned
func (h *Handlers) ListFindings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")

	var findings []recommendations.Finding

	var err error
	if status != "" {
		if ok := recommendations.FindingStatusIsValid(status); !ok {
			http.Error(w, "finding status must be 'active' or 'resolved'", http.StatusBadRequest)
			return
		}
		findings, err = h.FindingStorage.ListForStatus(status)
	} else {
		findings, err = h.FindingStorage.List()
	}
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	io.RespondJSON(ctx, h.Log, w, findings, http.StatusOK)
}

func (h *Handlers) GetFinding(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	findingID := chi.URLParam(r, "findingID")
	finding, err := h.FindingStorage.Get(findingID)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	if finding == nil {
		http.Error(w, "policy not found", http.StatusNotFound)
	} else {

		io.RespondJSON(ctx, h.Log, w, finding, http.StatusOK)
	}
}

func (h *Handlers) ListActionsForFinding(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	findingID := chi.URLParam(r, "findingID")
	alerts, err := h.ActionStorage.ListForPolicy(findingID)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}
	io.RespondJSON(ctx, h.Log, w, alerts, http.StatusOK)
}

type setPolicyStatusBody struct {
	Status string `json:"status"`
}

// FindFinding finds a finding by its role and status
func (h *Handlers) FindFinding(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	role := r.URL.Query().Get("role")
	status := r.URL.Query().Get("status")

	if role == "" || status == "" {
		http.Error(w, "role and status must be provided as query parameters", http.StatusBadRequest)
		return
	}

	finding, err := h.FindingStorage.FindByRole(storage.FindByRoleQuery{Role: role, Status: status})
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	if finding == nil {
		http.Error(w, "policy not found", http.StatusNotFound)
	} else {
		io.RespondJSON(ctx, h.Log, w, finding, http.StatusOK)
	}
}

func (h *Handlers) SetFindingStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	findingID := chi.URLParam(r, "findingID")

	var b setPolicyStatusBody

	if err := io.DecodeJSONBody(w, r, &b); err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	if ok := recommendations.FindingStatusIsValid(b.Status); !ok {
		http.Error(w, "finding status must be 'active' or 'resolved'", http.StatusBadRequest)
		return
	}

	finding, err := h.FindingStorage.Get(findingID)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	if finding == nil {
		http.Error(w, "finding not found", http.StatusNotFound)
		return
	}

	finding.Status = b.Status

	err = h.FindingStorage.CreateOrUpdate(*finding)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
	}

	io.RespondJSON(ctx, h.Log, w, finding, http.StatusOK)
}
