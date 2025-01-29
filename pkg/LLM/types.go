package LLM

import (
	"os"

	"github.com/duluk/ask-ollama/pkg/ollama"
)

type LLMConversations struct {
	Role            string `yaml:"role"`
	Content         string `yaml:"content"`
	Model           string `yaml:"model"`
	Timestamp       string `yaml:"timestamp"`
	NewConversation bool   `yaml:"new_conversation"`
	InputTokens     int32  `yaml:"input_tokens"`
	OutputTokens    int32  `yaml:"output_tokens"`
	ConvID          int    `yaml:"conv_id"`
}

type ClientResponse struct {
	Text         string
	InputTokens  int32
	OutputTokens int32
	MyEstInput   int32 // May be used at some point
}

type Client interface {
	Chat(args ClientArgs, termWidth int, tabWidth int) (ClientResponse, error)
}

type Ollama struct {
	APIKey string
	Client *ollama.Client
}

type ClientArgs struct {
	Model        *string
	Prompt       *string
	SystemPrompt *string
	Context      []LLMConversations
	MaxTokens    *int
	Temperature  *float32
	Log          *os.File
	ConvID       *int
}
