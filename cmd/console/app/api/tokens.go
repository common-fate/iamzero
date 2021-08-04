package api

import (
	"net/http"

	"github.com/common-fate/iamzero/api/io"
	"github.com/common-fate/iamzero/pkg/tokens"
	"github.com/go-chi/chi"
)

type ListTokensResponse struct {
	Tokens []tokens.Token `json:"tokens"`
}

func (h *Handlers) ListTokens(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tokens, err := h.TokenStore.List(ctx)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}
	res := ListTokensResponse{
		Tokens: tokens,
	}

	io.RespondJSON(ctx, h.Log, w, res, http.StatusOK)
}

func (h *Handlers) DeleteToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tokenID := chi.URLParam(r, "tokenID")
	err := h.TokenStore.Delete(ctx, tokenID)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type CreateTokenRequest struct {
	Name string `json:"name"`
}

func (h *Handlers) CreateToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var rec CreateTokenRequest
	if err := io.DecodeJSONBody(w, r, &rec); err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	token, err := h.TokenStore.Create(ctx, rec.Name)
	if err != nil {
		io.RespondError(ctx, h.Log, w, err)
		return
	}

	io.RespondJSON(ctx, h.Log, w, token, http.StatusOK)
}
