package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// Save original args to restore after each test
var (
	originalArgs []string
	originalHome string
)

func TestMain(m *testing.M) {
	originalArgs = os.Args
	originalHome = os.Getenv("HOME")

	// Run tests
	code := m.Run()

	// Restore HOME
	os.Args = originalArgs
	os.Setenv("HOME", originalHome)

	os.Exit(code)
}

func TestInitialize(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "ask-ai-test-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	os.Setenv("HOME", tmpHome)

	// Helper to reset flags and viper between tests
	reset := func() {
		pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
		viper.Reset()
	}

	// Helper to create a temporary config file
	createTempConfig := func(t *testing.T, content string) string {
		t.Helper()
		tmpfile, err := os.CreateTemp("", "config*.yml")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Remove(tmpfile.Name()) })
		return tmpfile.Name()
	}

	tests := []struct {
		name     string
		args     []string
		config   string // YAML content
		validate func(*testing.T, *Options, error)
	}{
		{
			name: "default values",
			args: []string{},
			validate: func(t *testing.T, opts *Options, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "ollama", opts.Model)
				assert.Equal(t, 512, opts.MaxTokens)
				assert.Equal(t, 2048, opts.ContextLength)
				assert.Equal(t, float32(0.7), opts.Temperature)
				assert.Equal(t, 0, opts.Context)
				assert.False(t, opts.ContinueChat)
			},
		},
		{
			name: "command line args override defaults",
			args: []string{
				"--model", "gemini",
				"--max-tokens", "1000",
				"--context-length", "8192",
				"--temperature", "0.9",
				"--continue",
			},
			validate: func(t *testing.T, opts *Options, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "gemini", opts.Model)
				assert.Equal(t, 1000, opts.MaxTokens)
				assert.Equal(t, 8192, opts.ContextLength)
				assert.Equal(t, float32(0.9), opts.Temperature)
				assert.True(t, opts.ContinueChat)
			},
		},
		{
			name: "config file values",
			args: []string{},
			config: `
model:
  default: "chatgpt"
  max_tokens: 3172
  temperature: 0.5
log:
  file: "/custom/log/path"
database:
  file: "/custom/db/path"
`,
			validate: func(t *testing.T, opts *Options, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "chatgpt", opts.Model)
				assert.Equal(t, 3172, opts.MaxTokens)
				assert.Equal(t, float32(0.5), opts.Temperature)
				assert.Equal(t, "/custom/log/path", opts.LogFileName)
				assert.Equal(t, "/custom/db/path", opts.DBFileName)
			},
		},
		{
			name: "command line args override config file",
			args: []string{
				"--model", "grok",
				"--max-tokens", "3000",
			},
			config: `
model:
  default: "chatgpt"
  max_tokens: 512
`,
			validate: func(t *testing.T, opts *Options, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "grok", opts.Model)
				assert.Equal(t, 3000, opts.MaxTokens)
			},
		},
		{
			name: "invalid config file",
			args: []string{"--config", "nonexistent.yml"},
			validate: func(t *testing.T, opts *Options, err error) {
				assert.Error(t, err) // Should not error on missing config
				assert.Contains(t, err.Error(), "error reading config file")
				assert.Nil(t, opts)
			},
		},
		{
			name: "context flag",
			args: []string{"--context", "5"},
			validate: func(t *testing.T, opts *Options, err error) {
				assert.NoError(t, err)
				assert.Equal(t, 5, opts.Context)
			},
		},
		{
			name: "system prompt",
			args: []string{"--system-prompt", "You are a helpful assistant"},
			validate: func(t *testing.T, opts *Options, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "You are a helpful assistant", opts.SystemPrompt)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state
			reset()

			// Store original args and restore them after the test
			defer func() {
				os.Args = originalArgs
			}()

			// Set up test args - use a fake program name as first arg
			os.Args = append([]string{"test-program"}, tt.args...)

			// Set up config file if provided
			if tt.config != "" {
				configPath := createTempConfig(t, tt.config)
				os.Args = append(os.Args, "--config", configPath)
			}

			// Run Initialize and validate results
			opts, err := Initialize()
			tt.validate(t, opts, err)
		})
	}
}

func HandleVersionFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
	}{
		{
			name:           "version flag",
			args:           []string{"--version"},
			expectedOutput: "ask-ai version: " + Version + "\n",
		},
		{
			name: "full version flag",
			args: []string{"--full-version"},
			expectedOutput: "Version: " + Version + "\n" +
				"Commit:  " + commit + "\n" +
				"Date:    " + date + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags and viper
			pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
			viper.Reset()

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Set up test args
			os.Args = append([]string{"test"}, tt.args...)

			// Run Initialize in a goroutine since it will call os.Exit
			done := make(chan bool)
			go func() {
				Initialize()
				done <- true
			}()

			// Close writer and restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			output := make([]byte, 1024)
			n, _ := r.Read(output)
			actual := string(output[:n])

			assert.Equal(t, tt.expectedOutput, actual)
		})
	}
}

func TestCustomConfigLocation(t *testing.T) {
	// Create a temporary directory for config
	tmpDir, err := os.MkdirTemp("", "ask-ai-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config file in custom location
	configContent := `
model:
  default: "custom-model"
  max_tokens: 1234
`
	configPath := filepath.Join(tmpDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Reset flags and viper
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	viper.Reset()

	// Set up test args
	os.Args = []string{"test", "--config", configPath}

	// Run Initialize
	opts, err := Initialize()
	assert.NoError(t, err)
	assert.Equal(t, "custom-model", opts.Model)
	assert.Equal(t, 1234, opts.MaxTokens)
}
