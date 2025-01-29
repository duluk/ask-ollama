package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"

	"github.com/duluk/ask-ollama/pkg/LLM"
	"github.com/duluk/ask-ollama/pkg/config"
	"github.com/duluk/ask-ollama/pkg/database"
)

// I'm probably writing "Ruby Go"...

func main() {
	var err error

	opts, err := config.Initialize()
	if err != nil {
		fmt.Println("Error initializing config: ", err)
		os.Exit(1)
	}

	if opts.DumpConfig {
		config.DumpConfig(opts)
	}

	err = os.MkdirAll(filepath.Dir(opts.LogFileName), 0755)
	if err != nil {
		fmt.Println("Error creating log directory: ", err)
		os.Exit(1)
	}

	var log_fd *os.File
	log_fd, err = os.OpenFile(opts.LogFileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("Error with chat log file: ", err)
	}
	defer log_fd.Close()

	// If DB exists, it just opens it; otherwise, it creates it first
	db, err := database.InitializeDB(opts.DBFileName, opts.DBTable)
	if err != nil {
		fmt.Println("Error opening database: ", err)
		os.Exit(1)
	}
	defer db.Close()

	model := opts.Model
	/* CONTEXT? LOAD IT */
	var promptContext []LLM.LLMConversations
	if opts.ConversationID != 0 {
		// The user may provide `--continue` along with `--id`, but that's fine
		// (and sensible). The intent is to load the one with the provided id.
		// promptContext, err = LLM.LoadConversationFromLog(log_fd,
		// opts.ConversationID)
		promptContext, err = db.LoadConversationFromDB(opts.ConversationID)
		if !pflag.CommandLine.Changed("model") {
			// model, _ = db.GetModel(opts.ConversationID)
			model = promptContext[len(promptContext)-1].Model
		}
		if err != nil {
			fmt.Println("Error loading conversation from log: ", err)
		}
	} else if opts.ContinueChat {
		promptContext, err = LLM.ContinueConversation(log_fd)
		if !pflag.CommandLine.Changed("model") {
			model = promptContext[len(promptContext)-1].Model
		}
		if err != nil {
			fmt.Println("Error reading log for continuing chat: ", err)
		}
	} else if opts.Context != 0 {
		promptContext, err = LLM.LastNChats(log_fd, opts.Context)
		if err != nil {
			fmt.Println("Error loading chat context from log: ", err)
		}
	}

	clientArgs := LLM.ClientArgs{
		Model:        &model,
		SystemPrompt: &opts.SystemPrompt,
		Context:      promptContext,
		MaxTokens:    &opts.MaxTokens,
		Temperature:  &opts.Temperature,
		Log:          log_fd,
	}

	// Make sure we are setting the correct conversation id when not provided
	if opts.ConversationID == 0 {
		clientArgs.ConvID = LLM.FindLastConversationID(log_fd)
		if clientArgs.ConvID == nil {
			// Most likely this is the first conversation
			clientArgs.ConvID = new(int)
			*clientArgs.ConvID = 0
		}
		if !opts.ContinueChat {
			(*clientArgs.ConvID)++
		}
	} else {
		clientArgs.ConvID = &opts.ConversationID
	}

	/* GET THE PROMPT */
	var prompt string
	if pflag.NArg() > 0 {
		prompt = pflag.Arg(0)
		clientArgs.Prompt = &prompt

		chatWithLLM(opts, clientArgs, db)
	} else {
		// Gracefully handle CTRL-C interrupt signal
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		go func() {
			<-sig
			fmt.Println("\nGoodbye!")
			os.Exit(0)
		}()

		for {
			prompt = getPromptFromUser(model)
			if prompt[0] == '/' {
				cmd := strings.Split(prompt, " ")[0]
				switch cmd {
				case "/help", "/?":
					fmt.Println("Special commands:")
					fmt.Println("  /exit: Exit the program")
					fmt.Println("  /context: Show the current context")
					fmt.Println("  /model <model>: Show the current model")
					fmt.Println("  /id: Show the current conversation ID")
					continue
				case "/exit", "/quit":
					fmt.Println("Goodbye!")
					os.Exit(0)
				case "/context":
					fmt.Println("Context: ", promptContext)
					continue
				case "/model":
					if len(prompt) > 6 && prompt[7:] != "" {
						model = prompt[7:]
						clientArgs.Model = &model
					}
					continue
				case "/id":
					fmt.Println("Conversation ID: ", *clientArgs.ConvID)
					continue
				}
			}
			clientArgs.Prompt = &prompt

			chatWithLLM(opts, clientArgs, db)

			opts.ContinueChat = true
			promptContext, err = LLM.ContinueConversation(log_fd)
			if err != nil {
				fmt.Println("Error reading log for continuing chat: ", err)
			}
			// TODO: promptContext will be nil if err != nil above. That's
			// probably what we want. Would write a test but not sure how to
			// test the LLM functions without using tokens.
			clientArgs.Context = promptContext
		}
	}
}

func chatWithLLM(opts *config.Options, args LLM.ClientArgs, db *database.ChatDB) {
	var client LLM.Client
	log := args.Log
	model := *args.Model
	continueChat := opts.ContinueChat

	switch model {
	case "ollama":
		client = LLM.NewOllama()
	default:
		fmt.Println("Unknown model: ", model)
		os.Exit(1)
	}
	LLM.LogChat(
		log,
		"User",
		*args.Prompt,
		"",
		continueChat,
		LLM.EstimateTokens(*args.Prompt),
		0,
		*args.ConvID,
	)

	fmt.Println("Assistant: ")
	resp, err := client.Chat(args, opts.ScreenWidth, opts.TabWidth)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
	fmt.Printf("\n\n-%s (convID: %d)\n", model, *args.ConvID)

	// If we want the timestamp in the log and in the database to match
	// exactly, we can set it here and pass it in to LogChat and
	// InsertConversation. As it stands, each function uses the current
	// timestamp when the function is executed.

	LLM.LogChat(
		log,
		"Assistant",
		resp.Text,
		model,
		continueChat,
		resp.InputTokens,
		resp.OutputTokens,
		*args.ConvID,
	)

	err = db.InsertConversation(
		*args.Prompt,
		resp.Text,
		model,
		*args.Temperature,
		resp.InputTokens,
		resp.OutputTokens,
		*args.ConvID,
	)
	if err != nil {
		fmt.Println("error inserting conversation into database: ", err)
	}
}

func getPromptFromUser(model string) string {
	fmt.Printf("%s> ", model)
	reader := bufio.NewReader(os.Stdin)
	prompt, err := reader.ReadString('\n')
	if err != nil {
		if err.Error() == "EOF" {
			fmt.Println("\nGoodbye!")
			os.Exit(0)
		}
		fmt.Println("Error reading prompt: ", err)
		os.Exit(1)
	}

	// Now clean up spaces and remove the newline we just captured
	prompt = strings.TrimSpace(prompt)
	if prompt[len(prompt)-1] == '\n' {
		prompt = prompt[:len(prompt)-1]
	}

	return prompt
}
