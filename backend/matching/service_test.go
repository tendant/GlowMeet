package matching

import (
	"context"
	"fmt"
	"glowmeet/xai"
	"sync"
	"testing"
	"time"
)

// Mock AI Client
type mockAIClient struct {
	mu       sync.Mutex
	response *xai.ChatResponse
	err      error
	calls    []xai.ChatRequest
}

func (m *mockAIClient) CreateChatCompletion(ctx context.Context, req xai.ChatRequest) (*xai.ChatResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, req)
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockAIClient) getCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func TestService_EndToEnd(t *testing.T) {
	mock := &mockAIClient{
		response: &xai.ChatResponse{
			Choices: []xai.Choice{
				{
					Message: xai.Message{
						Content: `{"score": 88.5, "reason": "Good match."}`,
					},
				},
			},
		},
	}

	service := NewServiceWithClient(mock)

	viewer := UserInput{ID: "v1", Summary: "Engineer", Interests: "Go, AI"}
	candidate := UserInput{ID: "c1", Summary: "Designer", Interests: "UI, AI"}

	// 1. Trigger Async Calculation
	service.CalculateMatchesAsync(viewer, []UserInput{candidate})

	// 2. Wait for worker to process (allow up to 1 second)
	success := false
	for i := 0; i < 20; i++ {
		match := service.GetMatch("v1", "c1")
		if match.Score > 0 {
			success = true
			if match.Score != 88.5 {
				t.Errorf("expected score 88.5, got %f", match.Score)
			}
			if match.Reason != "Good match." {
				t.Errorf("expected reason 'Good match.', got %s", match.Reason)
			}
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !success {
		t.Fatal("timed out waiting for match calculation")
	}

	// 3. Check Mock Calls
	count := mock.getCallCount()
	if count < 1 {
		t.Errorf("expected at least 1 mock call, got %d", count)
	}
}

func TestService_GetTopMatches(t *testing.T) {
	service := NewServiceWithClient(&mockAIClient{})

	// Manually inject data into cache
	service.updateCache("v1", "c1", MatchResult{TargetID: "c1", Score: 50.0})
	service.updateCache("v1", "c2", MatchResult{TargetID: "c2", Score: 90.0})
	service.updateCache("v1", "c3", MatchResult{TargetID: "c3", Score: 10.0})
	service.updateCache("v1", "c4", MatchResult{TargetID: "c4", Score: 75.0})

	matches := service.GetTopMatches("v1", 3)

	if len(matches) != 3 {
		t.Errorf("expected 3 matches, got %d", len(matches))
	}
	if matches[0].TargetID != "c2" {
		t.Errorf("expected top match c2, got %s", matches[0].TargetID)
	}
	if matches[1].TargetID != "c4" {
		t.Errorf("expected second match c4, got %s", matches[1].TargetID)
	}
	if matches[2].TargetID != "c1" {
		t.Errorf("expected third match c1, got %s", matches[2].TargetID)
	}
}

func TestService_CalculateEmpty(t *testing.T) {
	service := NewServiceWithClient(&mockAIClient{})
	// Should not crash
	service.CalculateMatchesAsync(UserInput{ID: "v1"}, []UserInput{})
}

func TestService_Concurrency(t *testing.T) {
	mock := &mockAIClient{
		response: &xai.ChatResponse{
			Choices: []xai.Choice{
				{Message: xai.Message{Content: `{"score": 50, "reason": "ok"}`}},
			},
		},
	}
	service := NewServiceWithClient(mock)

	var wg sync.WaitGroup

	// Writer goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			viewer := UserInput{ID: fmt.Sprintf("v%d", id), Interests: "x"}
			candidate := UserInput{ID: "c1", Interests: "y"}
			service.CalculateMatchesAsync(viewer, []UserInput{candidate})
		}(i)
	}

	// Reader goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := 0; k < 5; k++ {
				service.GetTopMatches("v1", 5)
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	wg.Wait()
	// Pass if no race/panic
}
