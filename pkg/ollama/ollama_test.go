package ollama

import (
	"encoding/json"
	"strconv"
	"testing"
)

// Deepseek R1 takes too long to think
const TestModel = "llama3.1"

func TestNewClient(t *testing.T) {
	client := NewClient("dummy-api-key")
	if client.APIKey != "dummy-api-key" {
		t.Errorf("Expected API key not set correctly")
	}
}

// This adds about 10-15 seconds to test time
func TestChatCompletion(t *testing.T) {
	req := ChatCompletionRequest{
		Model: TestModel,
		Messages: []Message{{
			Role:    "user",
			Content: "Resposne with the number 3",
		}},
	}

	testClient := NewClient("test-key")
	resp, err := testClient.ChatCompletion(req)

	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Error("Expected at least one completion choice")
	}
}

// Commenting out this test as we aren't validating with API keys (yet)
// func TestErrorHandling(t *testing.T) {
// 	// Create a test client with invalid API key
// 	invalidClient := NewClient("")
//
// 	_, err := invalidClient.ChatCompletion(ChatCompletionRequest{})
// 	if err != nil {
// 		if !strings.Contains(err.Error(), "failed to create request") {
// 			t.Errorf("Unexpected error: %v", err)
// 		}
// 		return
// 	}
//
// 	t.Fatalf("Expected API key error not thrown")
// }

func TestRateLimiting(t *testing.T) {
	client := NewClient("test-key")

	for i := 0; i < 5; i++ {
		req := ChatCompletionRequest{
			Model: TestModel,
			Messages: []Message{{
				Role:    "user",
				Content: "Test request " + strconv.Itoa(i),
			}},
		}
		_, err := client.ChatCompletion(req)
		if err != nil {
			t.Errorf("Unexpected error on request %d: %v", i, err)
		}
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	reqData := ChatCompletionRequest{
		Model: TestModel,
		Messages: []Message{{
			Role:    "user",
			Content: "Test message",
		}},
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var response ChatCompletionResponse
	if err := json.Unmarshal([]byte(jsonData), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
}
