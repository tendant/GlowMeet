package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"glowmeet/matching"
	"glowmeet/xai"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
)

type Config struct {
	Port          string
	ClientID      string
	ClientSecret  string
	RedirectURL   string
	AllowedOrigin string
	FrontendURL   string
	JWTSecret     string
	JWTTTL        time.Duration
	XAiAPIKey     string
	Persistence   string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	RedisTLS      bool
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
	config  *Config
	oauth   *oauth2.Config
	states  *stateStore
	users   UserStore
	tokens  tokenStore
	tweets  *tweetStore
	matcher *matching.Service
}

func main() {
	_ = godotenv.Load()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	srv := newServer(cfg)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("starting GlowMeet auth server on %s (redirect_url=%s, cors_origin=%s, frontend_url=%s, persistence=%s)", addr, cfg.RedirectURL, cfg.AllowedOrigin, cfg.FrontendURL, cfg.Persistence)
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
		FrontendURL:   getEnv("FRONTEND_URL", "/"),
		JWTSecret:     os.Getenv("APP_JWT_SECRET"),
		JWTTTL:        getEnvDuration("APP_JWT_TTL", 24*time.Hour),
		XAiAPIKey:     os.Getenv("XAI_API_KEY"),
		Persistence:   getEnv("PERSISTENCE", "memory"),
		RedisAddr:     getEnv("REDIS_ADDR", ""),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       getEnvInt("REDIS_DB", 0),
		RedisTLS:      getEnvBool("REDIS_TLS", false),
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
	if cfg.JWTSecret == "" {
		return nil, errors.New("missing APP_JWT_SECRET")
	}
	if cfg.Persistence != "memory" && cfg.Persistence != "redis" {
		cfg.Persistence = "memory"
	}

	return cfg, nil
}

func newServer(cfg *Config) *server {
	s := &server{
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
		states:  newStateStore(10 * time.Minute),
		users:   newUserStore(cfg),
		tokens:  newTokenStoreFromConfig(cfg),
		tweets:  newTweetStore(50),
		matcher: matching.NewService(cfg.XAiAPIKey, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB),
	}

	s.seedUsers()
	s.seedMatches()
	return s
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
		r.Post("/me", s.handleUpdateMe)
		r.Post("/me/location", s.handleUpdateLocation)
		r.Get("/users", s.handleUsers)
		r.Get("/users/{id}", s.handleUser)
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

	sessionID, err := randomString(32)
	if err != nil {
		logError(r, "failed creating session id", err)
		writeError(w, http.StatusInternalServerError, "session creation failed")
		return
	}

	profile, err := s.fetchXUser(ctx, token.AccessToken)
	if err != nil {
		logError(r, "failed fetching X profile after login", err)
	} else if profile.ID != "" {
		log.Printf("req_id=%s profile fetched login id=%s username=%s", middleware.GetReqID(r.Context()), profile.ID, profile.Username)
		s.users.upsert(profile)
		go s.fetchUserTweets(profile.ID, token.AccessToken) // This will trigger XAI analysis -> then trigger matching
	}

	s.tokens.upsert(profile.ID, tokenInfo{
		UserID:       profile.ID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	})

	sessionToken, err := s.issueJWT(profile.ID, token.Expiry)
	if err != nil {
		logError(r, "failed creating session token", err)
		writeError(w, http.StatusInternalServerError, "session creation failed")
		return
	}

	secureCookie := strings.HasPrefix(strings.ToLower(s.config.RedirectURL), "https")
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
		Expires:  token.Expiry,
	})

	redirectTarget := resolveRedirectTarget(s.config.FrontendURL)
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"session":        sessionID,
			"user":           profile,
			"session_expiry": token.Expiry,
		})
		return
	}

	http.Redirect(w, r, redirectTarget, http.StatusFound)
}

func (s *server) handleMe(w http.ResponseWriter, r *http.Request) {
	userID := s.resolveAccessToken(r)
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing access token")
		return
	}

	if userID == "" {
		writeError(w, http.StatusNotFound, "user not cached")
		return
	}

	profile, ok := s.users.get(userID)
	if !ok {
		// try to fetch using stored token
		if tok, ok := s.tokens.get(userID); ok && tok.AccessToken != "" {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()
			fresh, err := s.fetchXUser(ctx, tok.AccessToken)
			if err == nil && fresh.ID != "" {
				s.users.upsert(fresh)
				profile = fresh
				ok = true
			}
		}
		if !ok {
			writeError(w, http.StatusNotFound, "user not cached")
			return
		}
	}

	if profile.ID != "" {
		profile.Tweets = s.tweets.get(profile.ID)
	}

	writeJSON(w, http.StatusOK, profile)
}

func (s *server) handleUsers(w http.ResponseWriter, r *http.Request) {
	viewerID := s.resolveAccessToken(r)

	type userSummary struct {
		UserID        string   `json:"user_id"`
		Name          string   `json:"name,omitempty"`
		Username      string   `json:"username,omitempty"`
		ProfileImage  string   `json:"profile_image_url,omitempty"`
		Lat           float64  `json:"lat,omitempty"`
		Long          float64  `json:"long,omitempty"`
		MatchingScore float64  `json:"matching_score,omitempty"`
		MatchReason   string   `json:"match_reason,omitempty"`
		Summary       string   `json:"summary,omitempty"`
		Description   string   `json:"description,omitempty"`
		Tweets        []string `json:"tweets,omitempty"`
		Interests     string   `json:"interests,omitempty"`
	}

	var out []userSummary

	// 1. Try to get Top Matches if logged in
	if viewerID != "" {
		matches := s.matcher.GetTopMatches(viewerID, 5)
		if len(matches) > 0 {
			out = make([]userSummary, 0, len(matches))
			for _, m := range matches {
				u, ok := s.users.get(m.TargetID)
				if !ok {
					continue
				}
				tweets := s.tweets.get(u.ID)
				out = append(out, userSummary{
					UserID:        u.ID,
					Name:          u.Name,
					Username:      u.Username,
					ProfileImage:  u.ProfileImageURL,
					Lat:           u.Lat,
					Long:          u.Long,
					MatchingScore: m.Score,
					MatchReason:   m.Reason,
					Summary:       u.Summary,
					Description:   u.Description,
					Interests:     u.Interests,
					Tweets: func() []string {
						if len(tweets) > 0 {
							return []string{tweets[0]}
						}
						return nil
					}(),
				})
			}
		}
	}

	// 2. Fallback to default top 5 if no specific matches found
	if len(out) == 0 {
		users := s.users.top(5)
		out = make([]userSummary, 0, len(users))
		for _, u := range users {
			// Skip self if logged in (optional but good UI)
			if u.ID == viewerID {
				continue
			}
			tweets := s.tweets.get(u.ID)
			out = append(out, userSummary{
				UserID:        u.ID,
				Name:          u.Name,
				Username:      u.Username,
				ProfileImage:  u.ProfileImageURL,
				Lat:           u.Lat,
				Long:          u.Long,
				MatchingScore: u.MatchingScore,
				Summary:       u.Summary,
				Description:   u.Description,
				Interests:     u.Interests,
				Tweets: func() []string {
					if len(tweets) > 0 {
						return []string{tweets[0]}
					}
					return nil
				}(),
			})
		}
	}

	writeJSON(w, http.StatusOK, out)
}

func (s *server) handleUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "missing user id")
		return
	}

	user, ok := s.users.get(userID)
	if !ok {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	// Populate tweets from separate store
	if user.ID != "" {
		user.Tweets = s.tweets.get(user.ID)
	}

	// Calculate/Fetch Match Score if viewer is logged in
	viewerID := s.resolveAccessToken(r)

	// Define response structure that flattens userProfile fields
	// and adds an optional Match field.
	type userResponse struct {
		userProfile
		Match *matching.MatchResult `json:"match_info,omitempty"`
	}

	var match *matching.MatchResult
	if viewerID != "" && viewerID != user.ID {
		m := s.matcher.GetMatch(viewerID, user.ID)
		if m.Score > 0 {
			match = &m
		}
	}

	writeJSON(w, http.StatusOK, userResponse{
		userProfile: user,
		Match:       match,
	})
}

func (s *server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := s.resolveAccessToken(r)
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing access token")
		return
	}

	var body struct {
		Lat       float64 `json:"lat"`
		Long      float64 `json:"long"`
		Interests string  `json:"interests"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if len(body.Interests) > 512 {
		writeError(w, http.StatusBadRequest, "interests too long (max 512 chars)")
		return
	}

	s.users.updateProfile(userID, func(u userProfile) userProfile {
		if body.Interests != "" {
			u.Interests = body.Interests
		}
		return u
	})

	// Trigger XAI analysis to update summary/score based on new interests
	if body.Interests != "" {
		tweets := s.tweets.get(userID)
		if len(tweets) > 0 {
			go s.callXAIAnalysis(userID, tweets)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"interests": body.Interests,
	})
}

func (s *server) handleUpdateLocation(w http.ResponseWriter, r *http.Request) {
	userID := s.resolveAccessToken(r)
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing access token")
		return
	}

	var body struct {
		Lat  float64 `json:"lat"`
		Long float64 `json:"long"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if body.Lat == 0 && body.Long == 0 {
		writeError(w, http.StatusBadRequest, "lat/long required")
		return
	}

	s.users.updateLocation(userID, body.Lat, body.Long)
	writeJSON(w, http.StatusOK, map[string]any{
		"lat":  body.Lat,
		"long": body.Long,
	})
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
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Username        string   `json:"username"`
	ProfileImageURL string   `json:"profile_image_url,omitempty"`
	Lat             float64  `json:"lat,omitempty"`
	Long            float64  `json:"long,omitempty"`
	Summary         string   `json:"summary,omitempty"`
	BgImage         string   `json:"bg_image,omitempty"`
	Tweets          []string `json:"tweets,omitempty"`
	Interests       string   `json:"interests,omitempty"`
	MatchingScore   float64  `json:"matching_score,omitempty"`
	Description     string   `json:"description,omitempty"`
}

type UserStore interface {
	upsert(u userProfile)
	get(userID string) (userProfile, bool)
	top(n int) []userProfile
	getAllAsInputs() []matching.UserInput
	updateXAIData(userID, summary, imageURL string, score float64)
	updateLocation(userID string, lat, long float64)
	updateProfile(userID string, mutate func(userProfile) userProfile)
	loadFromFile(path string) error
	getRawMap() map[string]userProfile // helper for seeding logic access if needed, or refactor seeding
}

// memoryUserStore implementation
type memoryUserStore struct {
	mu   sync.Mutex
	lim  int
	data map[string]userProfile
}

func (s *memoryUserStore) getRawMap() map[string]userProfile {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data
}

type redisUserStore struct {
	client *redis.Client
}

func (s *redisUserStore) getRawMap() map[string]userProfile {
	return nil // not supported in redis mode effectively or implemented differently
}

type tokenInfo struct {
	UserID       string    `json:"user_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
}

type tokenStore interface {
	upsert(userID string, token tokenInfo)
	get(userID string) (tokenInfo, bool)
}

type memoryTokenStore struct {
	mu   sync.Mutex
	lim  int
	data map[string]tokenInfo
}

type redisTokenStore struct {
	client      *redis.Client
	ttlFallback time.Duration
}

type tweetStore struct {
	mu          sync.Mutex
	lim         int
	data        map[string][]string
	lastFetched map[string]time.Time
}

func newUserStore(cfg *Config) UserStore {
	if cfg.Persistence == "redis" && cfg.RedisAddr != "" {
		opts := &redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
			TLSConfig: func() *tls.Config {
				if cfg.RedisTLS {
					return &tls.Config{InsecureSkipVerify: false}
				}
				return nil
			}(),
		}
		client := redis.NewClient(opts)
		// No strict ping here to allow fallback logic in other places or lazy connect,
		// but consistent with token store, we return redis store.
		return &redisUserStore{client: client}
	}

	return &memoryUserStore{
		lim:  50,
		data: make(map[string]userProfile),
	}
}

func newMemoryTokenStore(limit int) *memoryTokenStore {
	return &memoryTokenStore{
		lim:  limit,
		data: make(map[string]tokenInfo),
	}
}

func newTokenStoreFromConfig(cfg *Config) tokenStore {
	if cfg.Persistence == "redis" && cfg.RedisAddr != "" {
		opts := &redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
			TLSConfig: func() *tls.Config {
				if cfg.RedisTLS {
					return &tls.Config{InsecureSkipVerify: false} // use defaults
				}
				return nil
			}(),
		}
		client := redis.NewClient(opts)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := client.Ping(ctx).Err(); err != nil {
			log.Printf("redis ping failed, falling back to memory: %v", err)
		} else {
			log.Printf("using redis persistence at %s db=%d tls=%t", cfg.RedisAddr, cfg.RedisDB, cfg.RedisTLS)
			return &redisTokenStore{client: client, ttlFallback: cfg.JWTTTL}
		}
	}

	return newMemoryTokenStore(200)
}

func newTweetStore(limit int) *tweetStore {
	return &tweetStore{
		lim:         limit,
		data:        make(map[string][]string),
		lastFetched: make(map[string]time.Time),
	}
}

func (s *memoryUserStore) upsert(u userProfile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u.MatchingScore == 0 {
		u.MatchingScore = defaultScore(u.ID)
	}
	if u.Description == "" && u.Username != "" {
		u.Description = fmt.Sprintf("X user @%s", u.Username)
	}
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

func (s *redisUserStore) upsert(u userProfile) {
	ctx := context.Background()
	data, _ := json.Marshal(u)
	s.client.Set(ctx, "user:"+u.ID, data, 0)
}

func (s *memoryUserStore) loadFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var users []userProfile
	if err := json.NewDecoder(f).Decode(&users); err != nil {
		return err
	}

	for _, u := range users {
		s.upsert(u)
	}
	return nil
}

func (s *redisUserStore) loadFromFile(path string) error {
	// Same logic, just calls upsert
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var users []userProfile
	if err := json.NewDecoder(f).Decode(&users); err != nil {
		return err
	}

	for _, u := range users {
		s.upsert(u)
	}
	return nil
}

func (s *memoryUserStore) top(n int) []userProfile {
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

func (s *redisUserStore) top(n int) []userProfile {
	// naive scan for now
	ctx := context.Background()
	keys, _ := s.client.Keys(ctx, "user:*").Result()
	out := []userProfile{}
	for _, k := range keys {
		val, _ := s.client.Get(ctx, k).Bytes()
		var u userProfile
		json.Unmarshal(val, &u)
		out = append(out, u)
		if len(out) >= n {
			break
		}
	}
	return out
}

func (s *memoryUserStore) getAllAsInputs() []matching.UserInput {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]matching.UserInput, 0, len(s.data))
	for _, u := range s.data {
		out = append(out, matching.UserInput{
			ID:        u.ID,
			Name:      u.Name,
			Username:  u.Username,
			Summary:   u.Summary,
			Interests: u.Interests,
		})
	}
	return out
}

func (s *redisUserStore) getAllAsInputs() []matching.UserInput {
	ctx := context.Background()
	keys, _ := s.client.Keys(ctx, "user:*").Result()
	out := []matching.UserInput{}
	for _, k := range keys {
		val, _ := s.client.Get(ctx, k).Bytes()
		var u userProfile
		json.Unmarshal(val, &u)
		out = append(out, matching.UserInput{
			ID:        u.ID,
			Name:      u.Name,
			Username:  u.Username,
			Summary:   u.Summary,
			Interests: u.Interests,
		})
	}
	return out
}

func (s *memoryUserStore) updateXAIData(userID, summary, imageURL string, score float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if user, ok := s.data[userID]; ok {
		user.Summary = summary
		if imageURL != "" {
			user.BgImage = imageURL
		}
		user.MatchingScore = score
		s.data[userID] = user
	}
}

func (s *redisUserStore) updateXAIData(userID, summary, imageURL string, score float64) {
	u, ok := s.get(userID)
	if ok {
		u.Summary = summary
		if imageURL != "" {
			u.BgImage = imageURL
		}
		u.MatchingScore = score
		s.upsert(u)
	}
}

func (s *memoryUserStore) updateLocation(userID string, lat, long float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if user, ok := s.data[userID]; ok {
		user.Lat = lat
		user.Long = long
		s.data[userID] = user
	}
}

func (s *redisUserStore) updateLocation(userID string, lat, long float64) {
	u, ok := s.get(userID)
	if ok {
		u.Lat = lat
		u.Long = long
		s.upsert(u)
	}
}

func (s *memoryUserStore) get(userID string) (userProfile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.data[userID]
	return user, ok
}

func (s *redisUserStore) get(userID string) (userProfile, bool) {
	ctx := context.Background()
	val, err := s.client.Get(ctx, "user:"+userID).Bytes()
	if err != nil {
		return userProfile{}, false
	}
	var u userProfile
	json.Unmarshal(val, &u)
	return u, true
}

func (s *memoryUserStore) updateProfile(userID string, mutate func(userProfile) userProfile) {
	if mutate == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.data[userID]
	if !ok {
		return
	}
	s.data[userID] = mutate(user)
}

func (s *redisUserStore) updateProfile(userID string, mutate func(userProfile) userProfile) {
	u, ok := s.get(userID)
	if ok {
		s.upsert(mutate(u))
	}
}

func defaultScore(id string) float64 {
	if id == "" {
		return 0
	}
	sum := sha256.Sum256([]byte(id))
	val := binary.BigEndian.Uint16(sum[:2])
	return float64(val%1000) / 10 // 0.0 - 99.9
}

func (s *memoryTokenStore) upsert(userID string, token tokenInfo) {
	if userID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[userID] = token
	if len(s.data) > s.lim {
		for k := range s.data {
			delete(s.data, k)
			if len(s.data) <= s.lim {
				break
			}
		}
	}
}

func (s *memoryTokenStore) get(userID string) (tokenInfo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.data[userID]
	return token, ok
}

func (s *redisTokenStore) upsert(userID string, token tokenInfo) {
	if userID == "" || s == nil || s.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ttl := time.Until(token.Expiry)
	if ttl <= 0 {
		ttl = s.ttlFallback
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	payload, err := json.Marshal(token)
	if err != nil {
		log.Printf("redis token marshal err: %v", err)
		return
	}
	if err := s.client.Set(ctx, redisTokenKey(userID), payload, ttl).Err(); err != nil {
		log.Printf("redis token set err: %v", err)
	}
}

func (s *redisTokenStore) get(userID string) (tokenInfo, bool) {
	if userID == "" || s == nil || s.client == nil {
		return tokenInfo{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	raw, err := s.client.Get(ctx, redisTokenKey(userID)).Bytes()
	if err != nil {
		if err != redis.Nil {
			log.Printf("redis token get err: %v", err)
		}
		return tokenInfo{}, false
	}
	var tok tokenInfo
	if err := json.Unmarshal(raw, &tok); err != nil {
		log.Printf("redis token unmarshal err: %v", err)
		return tokenInfo{}, false
	}
	return tok, true
}

func (s *tweetStore) set(userID string, tweets []string) {
	if userID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(tweets) > s.lim {
		tweets = tweets[:s.lim]
	}
	clone := append([]string(nil), tweets...)
	s.data[userID] = clone
	s.lastFetched[userID] = time.Now()
}

func (s *tweetStore) get(userID string) []string {
	if userID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tweets := s.data[userID]
	return append([]string(nil), tweets...)
}

func (s *tweetStore) shouldFetch(userID string, minInterval time.Duration) (bool, time.Time) {
	if userID == "" {
		return false, time.Time{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	last, ok := s.lastFetched[userID]
	if !ok {
		return true, time.Time{}
	}
	return time.Since(last) > minInterval, last
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

func (s *server) fetchUserTweets(userID, accessToken string) {
	if userID == "" || accessToken == "" {
		return
	}
	ok, last := s.tweets.shouldFetch(userID, 15*time.Minute)
	if !ok {
		if !last.IsZero() {
			log.Printf("fetch tweets skip user=%s recently_fetched=%s", userID, last.Format(time.RFC3339))
		}
		return
	}
	log.Printf("fetch tweets start user=%s", userID)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://api.twitter.com/2/users/%s/tweets?max_results=100&tweet.fields=created_at,text", userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("fetch tweets build request err for user=%s: %v", userID, err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("fetch tweets http err for user=%s: %v", userID, err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("fetch tweets failed user=%s status=%d body=%s", userID, resp.StatusCode, string(body))
		// mark a fetch attempt to avoid hammering when rate limited
		s.tweets.set(userID, s.tweets.get(userID))
		return
	}

	var payload struct {
		Data []struct {
			ID        string `json:"id"`
			Text      string `json:"text"`
			CreatedAt string `json:"created_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("fetch tweets unmarshal err for user=%s: %v", userID, err)
		return
	}

	texts := make([]string, 0, len(payload.Data))
	for _, t := range payload.Data {
		texts = append(texts, t.Text)
	}
	log.Printf("fetched %d tweets for user=%s", len(texts), userID)
	s.tweets.set(userID, texts)

	// call xai
	go s.callXAIAnalysis(userID, texts)
}

func (s *server) callXAIAnalysis(userID string, tweets []string) {
	if s.config.XAiAPIKey == "" {
		log.Printf("skipping xai analysis for user=%s: api key missing", userID)
		return
	}
	if len(tweets) == 0 {
		return
	}

	client := xai.NewClient(s.config.XAiAPIKey)

	// Combine first 50 tweets for context (to fit well within prompt limits while being comprehensive)
	limit := 50
	if len(tweets) < limit {
		limit = len(tweets)
	}
	contextText := strings.Join(tweets[:limit], "\n- ")

	// Fetch user interests
	var interests string
	if user, ok := s.users.get(userID); ok {
		interests = user.Interests
	}

	interestsContext := ""
	if interests != "" {
		interestsContext = fmt.Sprintf("\nThe user also has these stated interests: %s", interests)
	}

	prompt := fmt.Sprintf(`Analyze the following tweets from a user:%s
- %s

Generate a short 2-sentence summary of who they are. 
Also provide a 'matching score' from 0-100 indicating how socially engaging they seem based on their content and interests. 
Output purely JSON in the following format:
{"summary": "...", "score": 85.5}`, interestsContext, contextText)

	// Using CreateChatCompletion as we want JSON output which is easier with standard chat.
	// Ideally we'd use Structured Output if available, but here we'll parse the string.
	req := xai.ChatRequest{
		Model: xai.ModelGrok41Fast, // Use fast model for analysis
		Messages: []xai.Message{
			{Role: "user", Content: prompt},
		},
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		log.Printf("xai analysis failed for user=%s: %v", userID, err)
		return
	}

	if len(resp.Choices) == 0 {
		return
	}

	content := resp.Choices[0].Message.Content
	// Try to find JSON block if wrapped
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start != -1 && end != -1 && end > start {
		content = content[start : end+1]
	}

	var result struct {
		Summary string  `json:"summary"`
		Score   float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		log.Printf("xai analysis json parse failed for user=%s: %v content=%s", userID, err, content)
		return
	}

	log.Printf("xai analysis complete for user=%s: score=%.1f", userID, result.Score)

	// Generate AI Background Image based on summary
	var imageURL string
	if result.Summary != "" {
		imagePrompt := fmt.Sprintf("A cool, modernistic, abstract avatar representation of a matching persona described as: %s. Cyberpunk, vaporwave, or futuristic digital art style. High quality, vibrant colors, artistic, creative composition.", result.Summary)
		img, err := client.GenerateImage(context.Background(), imagePrompt)
		if err != nil {
			log.Printf("xai image generation failed for user=%s: %v", userID, err)
		} else {
			imageURL = img
			log.Printf("xai image generated for user=%s: %s", userID, imageURL)
		}
	}

	s.users.updateXAIData(userID, result.Summary, imageURL, result.Score)

	// After XAI analysis updates the user summary, trigger the Pairwise Matching.
	// This ensures we have the latest summary to compare against others.
	go s.triggerMatching(userID, tweets)
}

func (s *server) triggerMatching(userID string, userTweets []string) {
	candidates := s.users.getAllAsInputs()

	// Populate tweets for candidates (expensive loop map lookup but ok for 50 users)
	// Also create the 'primary' input
	var primary matching.UserInput

	// Fill tweets for candidates
	for i := range candidates {
		candidates[i].Tweets = s.tweets.get(candidates[i].ID)
		if candidates[i].ID == userID {
			primary = candidates[i]
			// Ensure primary has the tweets we just fetched/used
			if len(primary.Tweets) == 0 {
				primary.Tweets = userTweets
			}
		}
	}

	if primary.ID == "" {
		// Should have been in the list
		return
	}

	// Trigger background matching
	s.matcher.CalculateMatchesAsync(primary, candidates)
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

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if v == "1" || strings.ToLower(v) == "true" || strings.ToLower(v) == "yes" {
			return true
		}
		if v == "0" || strings.ToLower(v) == "false" || strings.ToLower(v) == "no" {
			return false
		}
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

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

func resolveRedirectTarget(frontendURL string) string {
	if frontendURL == "" {
		return "/"
	}
	lowered := strings.ToLower(frontendURL)
	if strings.HasPrefix(lowered, "http://") || strings.HasPrefix(lowered, "https://") {
		return frontendURL
	}
	if !strings.HasPrefix(frontendURL, "/") {
		return "/" + frontendURL
	}
	return frontendURL
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func redisTokenKey(userID string) string {
	return "token:" + userID
}

func (s *server) seedUsers() {
	if err := s.users.loadFromFile("data/users.json"); err != nil {
		log.Printf("warning: could not load fake users from data/users.json: %v", err)
	} else {
		// Populate tweetStore with seed data and trigger analysis
		// Since we are now behind an interface, we can't lock s.users.mu directly if it's the interface.
		// However, for seeding we know we just loaded data.
		// Let's rely on getAllAsInputs (which doesn't return tweets in current impl?) or just refactor.
		// Actually, loadFromFile calls upsert. so data is there.
		// We need to iterate all users to set tweets in tweetStore.

		// We can add a helper to iterate or just use a type assertion if we really want to optimize,
		// but let's just use a top-like scan or add a helper to the interface effectively.
		// For now, let's use the `top` method with a large number or just use getRawMap if we are in memory mode.
		// But in redis mode, loadFromFile puts them in redis.

		// Simple approach: get all users via getAllAsInputs (we need to update it to include Tweets or load them separately)
		// Wait, getAllAsInputs commented out tweets.
		// Let's just use the file data directly since we just read it!
		// That avoids all interface issues.
		f, err := os.Open("data/users.json")
		if err == nil {
			defer f.Close()
			var users []userProfile
			if err := json.NewDecoder(f).Decode(&users); err == nil {
				var usersToAnalyze []struct {
					ID     string
					Tweets []string
				}

				for _, u := range users {
					if len(u.Tweets) > 0 {
						s.tweets.set(u.ID, u.Tweets)
						usersToAnalyze = append(usersToAnalyze, struct {
							ID     string
							Tweets []string
						}{u.ID, u.Tweets})
					}
				}

				for _, u := range usersToAnalyze {
					go s.callXAIAnalysis(u.ID, u.Tweets)
				}
			}
		}
	}
}

func (s *server) seedMatches() {
	if err := s.matcher.LoadFromFile("data/matches.json"); err != nil {
		log.Printf("warning: could not load fake matches from data/matches.json: %v", err)
	}
}

func (s *server) resolveAccessToken(r *http.Request) string {
	sessionCookie, err := r.Cookie("access_token")
	if err != nil || sessionCookie.Value == "" {
		return ""
	}

	claims, err := s.parseJWT(sessionCookie.Value)
	if err != nil {
		logError(r, "invalid session token", err)
		return ""
	}

	return claims.Subject
}

func (s *server) issueJWT(userID string, fallbackExpiry time.Time) (string, error) {
	if userID == "" {
		return "", errors.New("missing user id for jwt")
	}

	exp := time.Now().Add(s.config.JWTTTL)
	if !fallbackExpiry.IsZero() && fallbackExpiry.Before(exp) {
		exp = fallbackExpiry
	}

	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(exp),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *server) parseJWT(tokenString string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token claims")
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			return parsed
		}
		if hours, err := strconv.Atoi(v); err == nil {
			return time.Duration(hours) * time.Hour
		}
	}
	return fallback
}
