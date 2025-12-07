package xai

import (
	"context"
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
