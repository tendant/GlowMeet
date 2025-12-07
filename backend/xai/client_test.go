package xai

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestClient_CreateChatCompletion_Integration(t *testing.T) {
	// Try to load .env from parent directory (backend root)
	_ = godotenv.Load("../.env")

	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("skipping integration test: XAI_API_KEY not set")
	}

	client := NewClient(apiKey)

	req := ChatRequest{
		Model: ModelGrok41Fast,
		Messages: []Message{
			{Role: "user", Content: "Hello, world!"},
		},
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateChatCompletion failed: %v", err)
	}

	if resp == nil {
		t.Fatal("response is nil")
	}

	if resp.ID == "" {
		t.Error("response ID is empty")
	}

	if len(resp.Choices) == 0 {
		t.Error("choices are empty")
	} else {
		if resp.Choices[0].Message.Content == "" {
			t.Error("first choice message content is empty")
		}
	}
}

func TestClient_GenerateImage_Integration(t *testing.T) {
	// Try to load .env from parent directory (backend root)
	_ = godotenv.Load("../.env")

	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("skipping integration test: XAI_API_KEY not set")
	}

	client := NewClient(apiKey)

	// Use a simple prompt for testing
	prompt := "A simple geometric shape, blue circle on white background"
	imageContent, err := client.GenerateImage(context.Background(), prompt)
	if err != nil {
		t.Fatalf("GenerateImage failed: %v", err)
	}

	if imageContent == "" {
		t.Error("generated image content is empty")
	}

	t.Logf("Generated Image Content: %s", imageContent)
}

func TestClient_GenerateResponse_WebSearch_Integration(t *testing.T) {
	// Try to load .env from parent directory (backend root)
	_ = godotenv.Load("../.env")

	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("skipping integration test: XAI_API_KEY not set")
	}

	client := NewClient(apiKey)

	req := ResponseRequest{
		Model: string(ModelGrok41Fast),
		Input: []Message{
			{Role: "user", Content: "What is xAI?"},
		},
		Tools: []ResponseTool{
			{
				Type: ToolTypeWebSearch,
				Filters: &WebSearchFilters{
					ExcludedDomains: []string{"wikipedia.org"},
				},
				EnableImageUnderstanding: true,
			},
		},
	}

	resp, err := client.GenerateResponse(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateResponse failed: %v", err)
	}

	if resp == nil {
		t.Fatal("response is nil")
	}

	if len(resp.Output) == 0 {
		t.Error("output is empty")
	} else {
		// Look for any text content
		foundContent := false
		for _, item := range resp.Output {
			if item.Content != nil {
				strContent := fmt.Sprintf("%v", item.Content)
				t.Logf("Response Content: %s", strContent)
				if strContent != "" {
					foundContent = true
					break
				}
			}
		}
		if !foundContent {
			t.Error("no text content found in output")
		}
	}
}

func TestClient_GenerateResponse_XSearch_Integration(t *testing.T) {
	// Try to load .env from parent directory (backend root)
	_ = godotenv.Load("../.env")

	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("skipping integration test: XAI_API_KEY not set")
	}

	client := NewClient(apiKey)

	req := ResponseRequest{
		Model: string(ModelGrok41Fast),
		Input: []Message{
			{Role: "user", Content: "Find tweets from user @tomlei90"},
		},
		Tools: []ResponseTool{
			{
				Type: ToolTypeXSearch,
			},
		},
	}

	resp, err := client.GenerateResponse(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateResponse failed: %v", err)
	}

	if resp == nil {
		t.Fatal("response is nil")
	}

	if len(resp.Output) == 0 {
		t.Error("output is empty")
	} else {
		// Look for any text content
		foundContent := false
		for _, item := range resp.Output {
			if item.Content != nil {
				strContent := fmt.Sprintf("%v", item.Content)
				t.Logf("Response Content: %s", strContent)
				if strContent != "" {
					foundContent = true
					break
				}
			}
		}
		if !foundContent {
			t.Error("no text content found in output")
		}
	}
}
