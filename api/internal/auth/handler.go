package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Handler struct {
	db          *pgxpool.Pool
	jwtSecret   string
	oauthConfig *oauth2.Config
	allowedSet  map[string]bool
}

func NewHandler(db *pgxpool.Pool, jwtSecret, clientID, clientSecret, redirectURL, allowedEmails string) *Handler {
	allowed := map[string]bool{}
	for _, e := range strings.Split(allowedEmails, ",") {
		e = strings.TrimSpace(strings.ToLower(e))
		if e != "" {
			allowed[e] = true
		}
	}

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	return &Handler{
		db:          db,
		jwtSecret:   jwtSecret,
		oauthConfig: cfg,
		allowedSet:  allowed,
	}
}

func (h *Handler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	state := randomState()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	url := h.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	token, err := h.oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "failed to exchange token", http.StatusInternalServerError)
		return
	}

	userInfo, err := fetchGoogleUserInfo(r.Context(), h.oauthConfig, token)
	if err != nil {
		log.Printf("fetch user info error: %v", err)
		http.Error(w, "failed to fetch user info", http.StatusInternalServerError)
		return
	}

	email := strings.ToLower(userInfo.Email)
	if !h.allowedSet[email] {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	user, err := upsertUser(r.Context(), h.db, userInfo)
	if err != nil {
		log.Printf("upsert user error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	jwtToken, err := h.issueJWT(user.ID, user.Email, user.Name)
	if err != nil {
		http.Error(w, "failed to issue token", http.StatusInternalServerError)
		return
	}

	frontendURL := r.URL.Query().Get("redirect")
	if frontendURL == "" {
		frontendURL = "/"
	}
	http.Redirect(w, r, frontendURL+"?token="+jwtToken, http.StatusTemporaryRedirect)
}

func (h *Handler) issueJWT(userID, email, name string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"name":  name,
		"exp":   time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

type googleUserInfo struct {
	Sub       string `json:"sub"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Picture   string `json:"picture"`
	Verified  bool   `json:"email_verified"`
}

func fetchGoogleUserInfo(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (*googleUserInfo, error) {
	client := cfg.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func upsertUser(ctx context.Context, db *pgxpool.Pool, info *googleUserInfo) (*userRow, error) {
	row := db.QueryRow(ctx,
		`INSERT INTO users (email, name, avatar_url)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name, avatar_url = EXCLUDED.avatar_url
		 RETURNING id, email, name, avatar_url`,
		strings.ToLower(info.Email), info.Name, info.Picture,
	)

	var u userRow
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL); err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return &u, nil
}

type userRow struct {
	ID        string
	Email     string
	Name      string
	AvatarURL string
}

func randomState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
