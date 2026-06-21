package auth

import (
	"fmt"
	"net/http"
	"strings"
)

// DevLoginHandler issues a JWT for any email without going through Google OAuth.
// Only enabled when DEV_AUTH=true — never compiled away, but the handler is only
// registered when the caller explicitly opts in.
func (h *Handler) DevLoginHandler(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("email")))
	if email == "" {
		email = "dev@local"
	}

	ctx := r.Context()
	user, err := upsertUser(ctx, h.db, &googleUserInfo{
		Email: email,
		Name:  "Dev User",
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("upsert user: %v", err), http.StatusInternalServerError)
		return
	}

	token, err := h.issueJWT(fmt.Sprintf("%v", user.ID), user.Email, user.Name)
	if err != nil {
		http.Error(w, "issue jwt", http.StatusInternalServerError)
		return
	}

	redirect := r.URL.Query().Get("redirect")
	if redirect == "" {
		redirect = "/"
	}
	http.Redirect(w, r, redirect+"?token="+token, http.StatusTemporaryRedirect)
}
