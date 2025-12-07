package matching

import (
	"context"
	"encoding/json"
	"fmt"
	"glowmeet/xai"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

type AIClient interface {
	CreateChatCompletion(ctx context.Context, req xai.ChatRequest) (*xai.ChatResponse, error)
}

// MatchResult represents a calculated compatibility score between two users.
type MatchResult struct {
	TargetID      string    `json:"target_id"`
	Score         float64   `json:"score"`
	Reason        string    `json:"reason"`
	SharedSummary string    `json:"shared_summary"`
	Timestamp     time.Time `json:"timestamp"`
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

	// Cache: Map[ViewerID] -> []MatchResult
	mu    sync.RWMutex
	cache map[string]map[string]MatchResult

	// Worker pool
	jobs chan matchingJob
}

type matchingJob struct {
	viewer    UserInput
	candidate UserInput
}

// NewService creates a new matching service with a background worker pool.
func NewService(apiKey string) *Service {
	return NewServiceWithClient(xai.NewClient(apiKey))
}

// NewServiceWithClient creates a new matching service with a provided AI client (useful for testing).
func NewServiceWithClient(client AIClient) *Service {
	s := &Service{
		aiClient: client,
		cache:    make(map[string]map[string]MatchResult),
		jobs:     make(chan matchingJob, 1000), // Buffer for pending calculations
	}

	// Start 5 background workers
	for i := 0; i < 5; i++ {
		go s.worker(i)
	}

	return s
}

// GetMatch returns a specific match result from cache. Returns empty if not found.
func (s *Service) GetMatch(viewerID, targetID string) MatchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if viewerDeps, ok := s.cache[viewerID]; ok {
		if match, ok := viewerDeps[targetID]; ok {
			return match
		}
	}
	return MatchResult{}
}

// GetTopMatches returns the top N matches for the viewer.
func (s *Service) GetTopMatches(viewerID string, n int) []MatchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	viewerDeps, ok := s.cache[viewerID]
	if !ok || len(viewerDeps) == 0 {
		return []MatchResult{}
	}

	// copy to slice
	matches := make([]MatchResult, 0, len(viewerDeps))
	for _, m := range viewerDeps {
		matches = append(matches, m)
	}

	// sort desc
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	if len(matches) > n {
		return matches[:n]
	}
	return matches
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
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.cache[viewerID]; !ok {
		s.cache[viewerID] = make(map[string]MatchResult)
	}
	s.cache[viewerID][targetID] = res
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
  "reason": "1 sentence explanation",
  "shared_summary": "1 engaging sentence summarizing user agreement/commonalities, addressing User A as 'You'. E.g. 'You two both love hiking!'"
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
		Score         float64 `json:"score"`
		Reason        string  `json:"reason"`
		SharedSummary string  `json:"shared_summary"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return MatchResult{}, err
	}

	return MatchResult{
		TargetID:      c.ID,
		Score:         out.Score,
		Reason:        out.Reason,
		SharedSummary: out.SharedSummary,
		Timestamp:     time.Now(),
	}, nil
}

func truncate(s []string, n int) []string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
