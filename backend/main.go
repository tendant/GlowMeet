package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

type Config struct {
	Port          string
	ClientID      string
	ClientSecret  string
	RedirectURL   string
	AllowedOrigin string
	FrontendURL   string
}

type stateEntry struct {
	verifier  string
	expiresAt time.Time
}

type stateStore struct {
	mu     sync.Mutex
	ttl    time.Duration
	values map[string]stateEntry
}

type server struct {
	config *Config
	oauth  *oauth2.Config
	states *stateStore
	users  *userStore
}

func main() {
	_ = godotenv.Load()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	srv := newServer(cfg)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("starting GlowMeet auth server on %s (redirect_url=%s, cors_origin=%s, frontend_url=%s)", addr, cfg.RedirectURL, cfg.AllowedOrigin, cfg.FrontendURL)
	if err := http.ListenAndServe(addr, srv.routes()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func loadConfig() (*Config, error) {
	cfg := &Config{
		Port:          getEnv("PORT", "8000"),
		ClientID:      os.Getenv("X_CLIENT_ID"),
		ClientSecret:  os.Getenv("X_CLIENT_SECRET"),
		RedirectURL:   os.Getenv("X_REDIRECT_URL"),
		AllowedOrigin: getEnv("CORS_ORIGIN", "*"),
		FrontendURL:   getEnv("FRONTEND_URL", "http://localhost:3000"),
	}

	if cfg.ClientID == "" {
		return nil, errors.New("missing X_CLIENT_ID")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("missing X_CLIENT_SECRET")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("missing X_REDIRECT_URL")
	}

	return cfg, nil
}

func newServer(cfg *Config) *server {
	return &server{
		config: cfg,
		oauth: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{"tweet.read", "users.read", "offline.access"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://twitter.com/i/oauth2/authorize",
				TokenURL: "https://api.twitter.com/2/oauth2/token",
			},
		},
		states: newStateStore(10 * time.Minute),
		users:  newUserStore(50),
	}
}

func (s *server) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestMetaLogger)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{s.config.AllowedOrigin},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/auth/x", func(r chi.Router) {
		r.Get("/login", s.handleXLogin)
		r.Get("/callback", s.handleXCallback)
	})

	r.Route("/api", func(r chi.Router) {
		r.Get("/me", s.handleMe)
		r.Get("/users", s.handleUsers)
	})

	return r
}

func (s *server) handleXLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randomString(32)
	if err != nil {
		logError(r, "state generation failed", err)
		writeError(w, http.StatusInternalServerError, "state generation failed")
		return
	}

	verifier, err := randomString(64)
	if err != nil {
		logError(r, "verifier generation failed", err)
		writeError(w, http.StatusInternalServerError, "verifier generation failed")
		return
	}

	challenge := pkceChallenge(verifier)
	s.states.put(state, verifier)
	log.Printf("req_id=%s login issued state=%s host=%s", middleware.GetReqID(r.Context()), state, r.Host)

	authURL := s.oauth.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	writeJSON(w, http.StatusOK, map[string]string{
		"authorization_url": authURL,
		"state":             state,
	})
}

func (s *server) handleXCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	if state == "" || code == "" {
		logError(r, "callback missing state or code", nil)
		writeError(w, http.StatusBadRequest, "missing state or code")
		return
	}

	verifier, ok := s.states.pop(state)
	if !ok {
		logError(r, "invalid or expired state", nil)
		writeError(w, http.StatusBadRequest, "invalid or expired state")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	token, err := s.oauth.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		logError(r, "token exchange failed", err)
		writeError(w, http.StatusBadGateway, fmt.Sprintf("token exchange failed: %v", err))
		return
	}

	profile, err := s.fetchXUser(ctx, token.AccessToken)
	if err != nil {
		logError(r, "failed fetching X profile after login", err)
	} else {
		s.users.upsert(profile)
	}

	secureCookie := strings.HasPrefix(strings.ToLower(s.config.RedirectURL), "https")
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    token.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
		Expires:  token.Expiry,
	})

	if token.RefreshToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    token.RefreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   secureCookie,
			SameSite: http.SameSiteLaxMode,
		})
	}

	http.Redirect(w, r, s.config.FrontendURL, http.StatusFound)
}

func (s *server) handleMe(w http.ResponseWriter, r *http.Request) {
	tokenCookie, err := r.Cookie("access_token")
	if err != nil || tokenCookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "missing access token")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	profile, err := s.fetchXUser(ctx, tokenCookie.Value)
	if err != nil {
		logError(r, "failed to fetch profile", err)
		writeError(w, http.StatusBadGateway, "failed to fetch profile")
		return
	}

	s.users.upsert(profile)
	writeJSON(w, http.StatusOK, profile)
}

func (s *server) handleUsers(w http.ResponseWriter, r *http.Request) {
	users := s.users.top(20)
	writeJSON(w, http.StatusOK, users)
}

func newStateStore(ttl time.Duration) *stateStore {
	return &stateStore{
		ttl:    ttl,
		values: make(map[string]stateEntry),
	}
}

func (s *stateStore) put(state, verifier string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	s.values[state] = stateEntry{
		verifier:  verifier,
		expiresAt: time.Now().Add(s.ttl),
	}
}

func (s *stateStore) pop(state string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	entry, ok := s.values[state]
	if !ok {
		return "", false
	}
	delete(s.values, state)
	if time.Now().After(entry.expiresAt) {
		return "", false
	}
	return entry.verifier, true
}

func (s *stateStore) cleanupLocked() {
	now := time.Now()
	for key, entry := range s.values {
		if now.After(entry.expiresAt) {
			delete(s.values, key)
		}
	}
}

func randomString(length int) (string, error) {
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

type userProfile struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	ProfileImageURL string `json:"profile_image_url,omitempty"`
}

type userStore struct {
	mu   sync.Mutex
	lim  int
	data map[string]userProfile
}

func newUserStore(limit int) *userStore {
	return &userStore{
		lim:  limit,
		data: make(map[string]userProfile),
	}
}

func (s *userStore) upsert(u userProfile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[u.ID] = u
	if len(s.data) > s.lim {
		// trim oldest by deleting arbitrary entries when limit exceeded
		for k := range s.data {
			delete(s.data, k)
			if len(s.data) <= s.lim {
				break
			}
		}
	}
}

func (s *userStore) top(n int) []userProfile {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]userProfile, 0, min(n, len(s.data)))
	for _, u := range s.data {
		result = append(result, u)
		if len(result) >= n {
			break
		}
	}
	return result
}

func (s *server) fetchXUser(ctx context.Context, accessToken string) (userProfile, error) {
	if accessToken == "" {
		return userProfile{}, errors.New("missing access token")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.twitter.com/2/users/me?user.fields=profile_image_url,name,username", nil)
	if err != nil {
		return userProfile{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return userProfile{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return userProfile{}, fmt.Errorf("x.com user fetch failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var payload struct {
		Data userProfile `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return userProfile{}, err
	}
	if payload.Data.ID == "" {
		return userProfile{}, errors.New("missing id in x.com response")
	}
	return payload.Data, nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func logError(r *http.Request, msg string, err error) {
	requestID := middleware.GetReqID(r.Context())
	prefix := fmt.Sprintf("req_id=%s %s %s host=%s", requestID, r.Method, r.URL.Path, r.Host)
	if err != nil {
		log.Printf("%s: %s: %v", prefix, msg, err)
		return
	}
	log.Printf("%s: %s", prefix, msg)
}

func requestMetaLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetReqID(r.Context())
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		log.Printf(
			"req_id=%s inbound scheme=%s host=%s path=%s proto=%s xfp=%s xff=%s ua=%q",
			requestID,
			scheme,
			r.Host,
			r.URL.Path,
			r.Proto,
			r.Header.Get("X-Forwarded-Proto"),
			r.Header.Get("X-Forwarded-For"),
			r.UserAgent(),
		)
		next.ServeHTTP(w, r)
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
