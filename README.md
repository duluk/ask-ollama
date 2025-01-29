# ask-ollama

## Description

The utility everyone has written for themselves, a basic CLI tool for asking LLMs questions without bothering with a mouse.

Full disclosure: this is my first Go project. I mainly write in Ruby, C and Python.

## Building

Go:

```bash
$ go mod tidy
$ go build cmd/ask-ollama/main.go
```

Or, as I'm doing now (bc I'm old):
```bash
$ make
```

## Installation

```bash
$ make install
```

## Usage

#### Set the API Key
1. Set {OPENAI,ANTHROPIC,GOOGLE,XAI}_API_KEY in your environment; or
1. Put the key in a file located at `$HOME/.config/ask-ollama/{openai,anthropic,google,xai}-api-key`

#### Ask a model a question
```bash
$ bin/ask-ollama "What is the best chess opening for a beginner?"
```

* If no query is provided, ask-ollama will prompt for one:
```
$ bin/ask-ollama
chatgpt> What is the best chess opening for a checkers player?
```

* You can provide a model with `--model <model>`:
```bash
$ bin/ask-ollama --model gemini "Why do you pull in so many modules for th Go API?"
```

* Continue the conversation
```bash
$ bin/ask-ollama --model grok "When is your knowledge cut-off?"
<...>
$ bin/ask-ollama --model grok --continue "So you're always mostly up to date?"
```

* Use last `n` queries for context:
```bash
$ bin/ask-ollama --context 3 "What are the last 3 things we talked about?"
```

* Search conversation history for a previous chat:
```bash
$ bin/ask-ollama --search "chess openings"
```

* Show a specific conversation:
```bash
$ bin/ask-ollama --show 3
```

* Continue a specific conversation:
```bash
$ bin/ask-ollama --id 42 "What about the Reti?"
```

### [NOTE]
> This is a work in progress and not all functionality has been added.
