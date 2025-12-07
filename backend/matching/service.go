package matching

import (
	"context"
	"encoding/json"
	"fmt"
	"glowmeet/xai"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type persistedMatch struct {
	ViewerID string  `json:"viewer_id"`
	TargetID string  `json:"target_id"`
	Score    float64 `json:"score"`
	Reason   string  `json:"reason"`
}

type AIClient interface {
	CreateChatCompletion(ctx context.Context, req xai.ChatRequest) (*xai.ChatResponse, error)
}

// MatchResult represents a calculated compatibility score between two users.
type MatchResult struct {
	TargetID  string    `json:"target_id"`
	Score     float64   `json:"score"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

// UserInput contains the necessary data for AI analysis.
type UserInput struct {
	ID        string
	Name      string
	Username  string
	Summary   string
	Interests string
	Tweets    []string
}

// Service handles pairwise matching logic.
type Service struct {
	aiClient AIClient

	// Storage driver
	storage Storage

	// Worker pool
	jobs chan matchingJob
}

type Storage interface {
	GetMatch(viewerID, targetID string) (MatchResult, bool)
	GetTopMatches(viewerID string, n int) []MatchResult
	UpdateMatch(viewerID, targetID string, res MatchResult)
	LoadFromFile(path string) error
}

type MemoryStorage struct {
	mu    sync.RWMutex
	cache map[string]map[string]MatchResult
}

func (s *MemoryStorage) GetMatch(viewerID, targetID string) (MatchResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if vDeps, ok := s.cache[viewerID]; ok {
		if m, ok := vDeps[targetID]; ok {
			return m, true
		}
	}
	return MatchResult{}, false
}

func (s *MemoryStorage) GetTopMatches(viewerID string, n int) []MatchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vDeps, ok := s.cache[viewerID]
	if !ok || len(vDeps) == 0 {
		return []MatchResult{}
	}
	matches := make([]MatchResult, 0, len(vDeps))
	for _, m := range vDeps {
		matches = append(matches, m)
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})
	if len(matches) > n {
		return matches[:n]
	}
	return matches
}

func (s *MemoryStorage) UpdateMatch(viewerID, targetID string, res MatchResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cache[viewerID]; !ok {
		s.cache[viewerID] = make(map[string]MatchResult)
	}
	s.cache[viewerID][targetID] = res
}

func (s *MemoryStorage) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var matches []persistedMatch
	if err := json.Unmarshal(data, &matches); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range matches {
		if _, ok := s.cache[m.ViewerID]; !ok {
			s.cache[m.ViewerID] = make(map[string]MatchResult)
		}
		s.cache[m.ViewerID][m.TargetID] = MatchResult{
			TargetID:  m.TargetID,
			Score:     m.Score,
			Reason:    m.Reason,
			Timestamp: time.Now(),
		}
	}
	return nil
}

type RedisStorage struct {
	client *redis.Client
}

func (s *RedisStorage) GetMatch(viewerID, targetID string) (MatchResult, bool) {
	ctx := context.Background()
	val, err := s.client.Get(ctx, fmt.Sprintf("match:%s:%s", viewerID, targetID)).Bytes()
	if err != nil {
		return MatchResult{}, false
	}
	var m MatchResult
	json.Unmarshal(val, &m)
	return m, true
}

func (s *RedisStorage) GetTopMatches(viewerID string, n int) []MatchResult {
	ctx := context.Background()
	// Get IDs from ZSET
	ids, err := s.client.ZRevRange(ctx, "matches:"+viewerID, 0, int64(n-1)).Result()
	if err != nil {
		return []MatchResult{}
	}
	out := make([]MatchResult, 0, len(ids))
	for _, id := range ids {
		// Parallel fetch or individual (individual for simplicity now)
		if m, ok := s.GetMatch(viewerID, id); ok {
			out = append(out, m)
		}
	}
	return out
}

func (s *RedisStorage) UpdateMatch(viewerID, targetID string, res MatchResult) {
	ctx := context.Background()
	data, _ := json.Marshal(res)

	pipe := s.client.Pipeline()
	// Store details
	pipe.Set(ctx, fmt.Sprintf("match:%s:%s", viewerID, targetID), data, 0)
	// Update ranking
	pipe.ZAdd(ctx, "matches:"+viewerID, redis.Z{Score: res.Score, Member: targetID})
	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Printf("[matcher] redis update error: %v", err)
	}
}

func (s *RedisStorage) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var matches []persistedMatch
	if err := json.Unmarshal(data, &matches); err != nil {
		return err
	}
	for _, m := range matches {
		s.UpdateMatch(m.ViewerID, m.TargetID, MatchResult{
			TargetID:  m.TargetID,
			Score:     m.Score,
			Reason:    m.Reason,
			Timestamp: time.Now(),
		})
	}
	return nil
}

type matchingJob struct {
	viewer    UserInput
	candidate UserInput
}

// NewService creates a new matching service with a background worker pool.
func NewService(apiKey string, redisAddr, redisPwd string, redisDB int) *Service {
	client := xai.NewClient(apiKey)
	var storage Storage
	if redisAddr != "" {
		storage = &RedisStorage{
			client: redis.NewClient(&redis.Options{
				Addr:     redisAddr,
				Password: redisPwd,
				DB:       redisDB,
			}),
		}
		log.Printf("[matcher] using redis storage")
	} else {
		storage = &MemoryStorage{
			cache: make(map[string]map[string]MatchResult),
		}
		log.Printf("[matcher] using memory storage")
	}

	s := &Service{
		aiClient: client,
		storage:  storage,
		jobs:     make(chan matchingJob, 1000),
	}
	for i := 0; i < 5; i++ {
		go s.worker(i)
	}
	return s
}

// NewServiceWithClient creates a new matching service with a provided AI client (useful for testing).
// It defaults to MemoryStorage.
func NewServiceWithClient(client AIClient) *Service {
	s := &Service{
		aiClient: client,
		storage: &MemoryStorage{
			cache: make(map[string]map[string]MatchResult),
		},
		jobs: make(chan matchingJob, 1000),
	}
	for i := 0; i < 5; i++ {
		go s.worker(i)
	}
	return s
}

// LoadFromFile loads pre-calculated matches from a JSON file.
func (s *Service) LoadFromFile(path string) error {
	return s.storage.LoadFromFile(path)
}

// GetMatch returns a specific match result from cache. Returns empty if not found.
func (s *Service) GetMatch(viewerID, targetID string) MatchResult {
	if m, ok := s.storage.GetMatch(viewerID, targetID); ok {
		return m
	}
	return MatchResult{}
}

// GetTopMatches returns the top N matches for the viewer.
func (s *Service) GetTopMatches(viewerID string, n int) []MatchResult {
	return s.storage.GetTopMatches(viewerID, n)
}

// CalculateMatchesAsync queues jobs to calculate matches between the primary user and all candidates.
func (s *Service) CalculateMatchesAsync(primary UserInput, candidates []UserInput) {
	go func() {
		for _, c := range candidates {
			if c.ID == primary.ID {
				continue
			}
			s.jobs <- matchingJob{viewer: primary, candidate: c}
			// Queue reverse direction too if symmetric (optional, but good for UX)
			s.jobs <- matchingJob{viewer: c, candidate: primary}
		}
	}()
}

func (s *Service) worker(id int) {
	for job := range s.jobs {
		// 1. Check if we already have a recent result (e.g. < 24h) to skip re-work
		// (For simplicity in this step, we'll overwrite if queued)

		// 2. Call AI
		res, err := s.callAI(job.viewer, job.candidate)
		if err != nil {
			log.Printf("[matcher] worker %d failed: %v", id, err)
			continue
		}

		// 3. Update Cache
		s.updateCache(job.viewer.ID, job.candidate.ID, res)
	}
}

func (s *Service) updateCache(viewerID, targetID string, res MatchResult) {
	s.storage.UpdateMatch(viewerID, targetID, res)
}

func (s *Service) callAI(v, c UserInput) (MatchResult, error) {
	// If no data, skip
	if len(v.Tweets) == 0 && v.Interests == "" {
		return MatchResult{}, fmt.Errorf("viewer has no data")
	}

	prompt := fmt.Sprintf(`Analyze social compatibility between User A and User B.
User A: %s. Interests: %s. Recent tweets: %s.
User B: %s. Interests: %s. Recent tweets: %s.

Return JSON: {
  "score": 0-100, 
  "reason": "Very brief sentence on why they are a good match. Address User A as 'You'. E.g. 'You both love hiking and outdoor adventures!'"
}`,
		v.Summary, v.Interests, strings.Join(truncate(v.Tweets, 5), " | "),
		c.Summary, c.Interests, strings.Join(truncate(c.Tweets, 5), " | "))

	req := xai.ChatRequest{
		Model: xai.ModelGrok41Fast,
		Messages: []xai.Message{
			{Role: "user", Content: prompt},
		},
	}

	resp, err := s.aiClient.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return MatchResult{}, err
	}
	if len(resp.Choices) == 0 {
		return MatchResult{}, fmt.Errorf("no choices")
	}

	content := resp.Choices[0].Message.Content
	// Simple JSON extraction
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start != -1 && end != -1 && end > start {
		content = content[start : end+1]
	}

	var out struct {
		Score  float64 `json:"score"`
		Reason string  `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return MatchResult{}, err
	}

	return MatchResult{
		TargetID:  c.ID,
		Score:     out.Score,
		Reason:    out.Reason,
		Timestamp: time.Now(),
	}, nil
}

func truncate(s []string, n int) []string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
