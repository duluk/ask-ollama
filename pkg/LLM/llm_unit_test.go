package LLM

import (
	"fmt"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
)

func TestLogChat(t *testing.T) {
	const contChat = true
	tempFile, err := os.CreateTemp("", "test_log_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	err = LogChat(tempFile, "user", "Hello", "gpt-3.5-turbo", contChat, 112, 420, 3)
	if err != nil {
		t.Errorf("LogChat failed: %v", err)
	}

	conversations, err := LoadChatLog(tempFile)
	if err != nil {
		t.Errorf("Failed to load chat log: %v", err)
	}

	if len(conversations) != 1 {
		t.Errorf("Expected 1 conversation, got %d", len(conversations))
	}

	if conversations[0].Role != "user" ||
		conversations[0].Content != "Hello" ||
		conversations[0].Model != "gpt-3.5-turbo" ||
		conversations[0].NewConversation != !contChat ||
		conversations[0].InputTokens != 112 ||
		conversations[0].OutputTokens != 420 ||
		conversations[0].ConvID != 3 {
		t.Errorf("Conversation data mismatch")
	}
}

func TestLoadChatLog(t *testing.T) {
	const contChat = true
	tempFile, err := os.CreateTemp("", "test_load_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	testData := []LLMConversations{
		{
			Role:            "user",
			Content:         "Hello",
			Model:           "gpt-3.5-turbo",
			Timestamp:       time.Now().Format(time.RFC3339),
			NewConversation: !contChat,
			InputTokens:     112,
			OutputTokens:    420,
			ConvID:          3,
		},
	}

	yamlData, _ := yaml.Marshal(testData)
	tempFile.Write(yamlData)

	conversations, err := LoadChatLog(tempFile)
	if err != nil {
		t.Errorf("LoadChatLog failed: %v", err)
	}

	if len(conversations) != 1 {
		t.Errorf("Expected 1 conversation, got %d", len(conversations))
	}

	if conversations[0].Role != "user" ||
		conversations[0].Content != "Hello" ||
		conversations[0].Model != "gpt-3.5-turbo" ||
		conversations[0].NewConversation != !contChat ||
		conversations[0].InputTokens != 112 ||
		conversations[0].OutputTokens != 420 ||
		conversations[0].ConvID != 3 {
		t.Errorf("Conversation data mismatch")
	}
}

func TestLastNChats(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_last_n_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	testData := []LLMConversations{
		{Role: "user", Content: "Hello", Model: "gpt-3.5-turbo"},
		{Role: "assistant", Content: "Hi there!", Model: "gpt-3.5-turbo"},
		{Role: "user", Content: "How are you?", Model: "gpt-3.5-turbo"},
	}

	yamlData, _ := yaml.Marshal(testData)
	tempFile.Write(yamlData)

	conversations, err := LastNChats(tempFile, 2)
	if err != nil {
		t.Errorf("LastNChats failed: %v", err)
	}

	if len(conversations) != 2 {
		t.Errorf("Expected 2 conversations, got %d", len(conversations))
	}

	if conversations[0].Content != "Hi there!" || conversations[1].Content != "How are you?" {
		t.Errorf("Incorrect last N chats returned")
	}
}

func TestContinueConversation(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_continue_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	testData := []LLMConversations{
		{Role: "User", Content: "Hello", Model: "gpt-3.5-turbo", NewConversation: true},
		{Role: "Assistant", Content: "Hi there!", Model: "gpt-3.5-turbo", NewConversation: true},
		{Role: "User", Content: "How are you?", Model: "gpt-3.5-turbo", NewConversation: true},
		{Role: "Assistant", Content: "I'm doing well, thanks!", Model: "gpt-3.5-turbo", NewConversation: true},
	}

	yamlData, _ := yaml.Marshal(testData)
	tempFile.Write(yamlData)

	conversations, err := ContinueConversation(tempFile)
	if err != nil {
		t.Errorf("ContinueConversation failed: %v", err)
	}

	if len(conversations) != 2 {
		t.Errorf("Expected 2 conversations, got %d", len(conversations))
	}

	if conversations[0].Content != "How are you?" || conversations[1].Content != "I'm doing well, thanks!" {
		t.Errorf("Incorrect conversation continuation")
	}
}

func TestLoadConversationFromLog(t *testing.T) {
	tempFile, err := os.CreateTemp("", "testlog")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	testData := []LLMConversations{
		{
			Role:            "user",
			Content:         "Hello",
			Model:           "gpt-3.5-turbo",
			Timestamp:       time.Now().Format(time.RFC3339),
			NewConversation: false,
			InputTokens:     112,
			OutputTokens:    420,
			ConvID:          1,
		},
		{
			Role:            "user",
			Content:         "Hi",
			Model:           "gpt-3.5-turbo",
			Timestamp:       time.Now().Format(time.RFC3339),
			NewConversation: false,
			InputTokens:     112,
			OutputTokens:    420,
			ConvID:          2,
		},
		{
			Role:            "assistant",
			Content:         "Hello here!",
			Model:           "gpt-3.5-turbo",
			Timestamp:       time.Now().Format(time.RFC3339),
			NewConversation: false,
			InputTokens:     112,
			OutputTokens:    420,
			ConvID:          1,
		},
	}

	yamlData, _ := yaml.Marshal(testData)
	tempFile.Write(yamlData)

	if _, err := tempFile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek to beginning of file: %v", err)
	}

	tests := []struct {
		name        string
		convID      int
		expectedLen int
		expectError bool
	}{
		{"Existing conversation", 1, 2, false},
		{"Non-existing conversation", 3, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convs, err := LoadConversationFromLog(tempFile, tt.convID)
			fmt.Printf("convs: %v\n", convs)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if len(convs) != tt.expectedLen {
				t.Errorf("Expected %d conversations, got %d", tt.expectedLen, len(convs))
			}

			for _, conv := range convs {
				if conv.ConvID != tt.convID {
					t.Errorf("Expected ConvID %d, got %d", tt.convID, conv.ConvID)
				}
			}
		})
	}

	t.Run("Invalid file", func(t *testing.T) {
		invalidFile, _ := os.CreateTemp("", "invalidlog")
		defer os.Remove(invalidFile.Name())
		defer invalidFile.Close()

		_, err := LoadConversationFromLog(invalidFile, 1)
		if err == nil {
			t.Errorf("Expected error for invalid file, got nil")
		}
	})
}

func TestFindLastConversationID(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "testchatlog")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	testCases := []struct {
		name     string
		content  string
		expected *int
	}{
		{
			name:     "Empty log",
			content:  "",
			expected: nil,
		},
		{
			name: "Single conversation",
			content: `[
			{"conv_id": 1, "Role": "user", "Content": "Hello"},
            ]`,
			expected: intPtr(1),
		},
		{
			name: "Multiple conversations",
			content: `[
			{"conv_id": 1, "Role": "user", "Content": "Hello"},
			{"conv_id": 3, "Role": "user", "Content": "Hi"},
			{"conv_id": 2, "Role": "user", "Content": "Hi"}
            ]`,
			expected: intPtr(3),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tmpfile.WriteString(tc.content); err != nil {
				t.Fatalf("Failed to write to temporary file: %v", err)
			}
			tmpfile.Seek(0, 0) // Reset file pointer to beginning

			result := FindLastConversationID(tmpfile)

			if tc.expected == nil && result != nil {
				t.Errorf("Expected nil, but got %d", *result)
			} else if tc.expected != nil && result == nil {
				t.Errorf("Expected %d, but got nil", *tc.expected)
			} else if tc.expected != nil && result != nil && *tc.expected != *result {
				t.Errorf("Expected %d, but got %d", *tc.expected, *result)
			}

			tmpfile.Truncate(0)
			tmpfile.Seek(0, 0)
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func TestGetClientKey(t *testing.T) {
	// Test environment variable (first option)
	os.Setenv("TEST_API_KEY", "test-key")
	key := getClientKey("test")
	if key != "test-key" {
		t.Errorf("Expected 'test-key', got '%s'", key)
	}
	os.Unsetenv("TEST_API_KEY")

	home := os.Getenv("HOME")
	os.MkdirAll(home+"/.config/ask-ai", 0755)
	os.WriteFile(home+"/.config/ask-ai/test-api-key", []byte("file-test-key"), 0644)
	defer os.Remove(home + "/.config/ask-ai/test-api-key")

	key = getClientKey("test")
	if key != "file-test-key" {
		t.Errorf("Expected 'file-test-key', got '%s'", key)
	}
}
