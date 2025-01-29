package LLM

import (
	"os"
	"strings"

	"github.com/duluk/ask-ollama/pkg/linewrap"
	"github.com/duluk/ask-ollama/pkg/ollama"
)

func NewOllama() *Ollama {
	apiKey := ""
	client := ollama.NewClient(apiKey)

	return &Ollama{APIKey: apiKey, Client: client}
}

func (cs *Ollama) Chat(args ClientArgs, termWidth int, tabWidth int) (ClientResponse, error) {
	client := cs.Client

	var msgCtx string

	for _, msg := range args.Context {
		msg.Role = strings.ToLower(msg.Role)
		switch msg.Role {
		case "user":
			msgCtx += "User: " + msg.Content + "\n"
		case "assistant":
			msgCtx += "Assistant: " + msg.Content + "\n"
		}
	}

	const minTokens = 32768
	const OllamaModelDeepseekR1_8b = "deepseek-r1:8b"
	const OllamaModelDeepseekR1_14b = "deepseek-r1:14b"

	myInputEstimate := EstimateTokens(msgCtx + *args.Prompt + *args.SystemPrompt)
	adjustedMaxTokens := int(myInputEstimate + int32(*args.MaxTokens))
	req := ollama.ChatCompletionRequest{
		Model: OllamaModelDeepseekR1_14b,
		Messages: []ollama.Message{
			{
				Role:    "system",
				Content: *args.SystemPrompt,
			},
			{
				Role:    "assistant",
				Content: msgCtx,
			},
			{
				Role:    "user",
				Content: *args.Prompt,
			},
		},
		// Since this is a local model, let's give it some room to cook
		MaxTokens:   max(adjustedMaxTokens, minTokens),
		Temperature: float64(*args.Temperature),
	}

	resp, err := client.ChatCompletion(req)
	if err != nil {
		return ClientResponse{}, err
	}

	usage := resp.Usage

	respText := resp.Choices[0].Message.Content
	wrapper := linewrap.NewLineWrapper(termWidth, tabWidth, os.Stdout)

	if _, err := wrapper.Write([]byte(respText)); err != nil {
		return ClientResponse{}, err
	}

	return ClientResponse{
		Text:         respText,
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
	}, nil
}
