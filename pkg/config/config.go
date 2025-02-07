package config

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/term"

	"gopkg.in/yaml.v3"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/duluk/ask-ollama/pkg/database"
)

const Version = "0.0.1"

const MaxTermWidth = 80
const widthPad = 5
const TabWidth = 4

var (
	commit = "Unknown"
	date   = "Unknown"
)

type Config struct {
	General  GeneralConfig    `mapstructure:"general"`
	Models   map[string]Model `mapstructure:"models"`
	Logging  LogConfig        `mapstructure:"logging"`
	Database DBConfig         `mapstructure:"database"`
	Roles    map[string]Role  `mapstructure:"roles"`
	Opts     Options
}

type GeneralConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

type Model struct {
	Name             string  `mapstructure:"name"`
	MaxTokens        int     `mapstructure:"max_tokens"`
	Temperature      float64 `mapstructure:"temperature"`
	TopP             float64 `mapstructure:"top_p"`
	PresencePenalty  float64 `mapstructure:"presence_penalty,omitempty"`
	FrequencyPenalty float64 `mapstructure:"frequency_penalty,omitempty"`
	Timeout          int     `mapstructure:"timeout"`
}

type LogConfig struct {
	LogFile     string `mapstructure:"log_file"`
	LogLevel    string `mapstructure:"log_level"`
	MaxFileSize int    `mapstructure:"max_file_size"`
	BackupCount int    `mapstructure:"backup_count"`
}

type DBConfig struct {
	Type           string `mapstructure:"type"`
	Path           string `mapstructure:"path"`
	TableName      string `mapstructure:"table_name"`
	BackupInterval int    `mapstructure:"backup_interval"`
}

type Role struct {
	Description string `mapstructure:"description"`
	Prompt      string `mapstructure:"prompt"`
}

type Options struct {
	Model string
	// Context        int
	// ContextLength  int
	ContinueChat   bool
	DumpConfig     bool
	ConversationID int
	ScreenWidth    int
	ScreenHeight   int
	TabWidth       int
}

func (c *Config) String() string {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func Initialize() (*Config, error) {
	configDir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "ask-ollama")
	if configDir == "" {
		configDir = filepath.Join(os.Getenv("HOME"), ".config", "ask-ollama")
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(configDir)

	pflag.StringP("config", "C", "", "Configuration file")
	pflag.StringP("model", "m", "", "Model to use")
	pflag.IntP("id", "i", 0, "Conversation ID")
	pflag.IntP("show", "s", 0, "Show conversation")
	pflag.BoolP("continue", "c", false, "Continue conversation")
	pflag.BoolP("version", "v", false, "Show version")
	pflag.BoolP("full-version", "V", false, "Show full version")
	pflag.BoolP("dump-config", "d", false, "Dump configuration")

	pflag.Parse()

	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return nil, fmt.Errorf("error binding flags: %w", err)
	}

	viper.SetDefault("model", "deepseek-r1")
	viper.SetDefault("logging.log_file", filepath.Join(configDir, "ask-ollama.chat.yml"))
	viper.SetDefault("database.path", filepath.Join(configDir, "ask-ollama.db"))
	viper.SetDefault("database.table_name", "conversations")
	// viper.SetDefault("screen.width", width)
	// viper.SetDefault("screen.height", height)

	if configFile := viper.GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file (%s): %w", viper.ConfigFileUsed(), err)
		}
		fmt.Printf("Config file read: %s\n", viper.ConfigFileUsed())
	}

	// Setup environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ASKOLLAMA")

	var config Config
	decoderConfig := viper.DecoderConfigOption(func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "mapstructure"
		// This allows "8080" and 8080 to both be decoded into an int:
		dc.WeaklyTypedInput = true
	})

	if err := viper.Unmarshal(&config, decoderConfig); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	config.Logging.LogFile = os.ExpandEnv(config.Logging.LogFile)
	config.Database.Path = os.ExpandEnv(config.Database.Path)

	// fmt.Printf("Config loaded: %+v\n", config)

	config.Opts.Model = viper.GetString("model")
	config.Opts.ConversationID = viper.GetInt("id")
	config.Opts.ContinueChat = viper.GetBool("continue")
	config.Opts.ScreenWidth, config.Opts.ScreenHeight = determineScreenSize()
	config.Opts.TabWidth = TabWidth
	config.Opts.DumpConfig = viper.GetBool("dump-config")

	// fmt.Printf("Config dump: %+v\n", config)

	// Handle version flags and bail if necessary
	if handleVersionFlags() {
		os.Exit(0)
	}

	// if viper.GetString("search") != "" {
	// 	searchForConversation(viper.GetString("search"))
	// }

	if viper.GetInt("show") != 0 {
		showConversation(viper.GetInt("show"))
	}

	return &config, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	return -min(-a, -b)
}

// Maybe this shouldn't be in config...
func searchForConversation(search string) {
	if viper.GetString("database.file") == "" {
		fmt.Println("Database file not set")
		os.Exit(1)
	}
	if viper.GetString("database.table") == "" {
		fmt.Println("Database table not set")
		os.Exit(1)
	}

	db, err := database.InitializeDB(os.ExpandEnv(viper.GetString("database.file")), viper.GetString("database.table"))
	if err != nil {
		fmt.Printf("Error opening database: %s", err)
		os.Exit(1)
	}
	defer db.Close()

	ids, err := db.SearchForConversation(search)
	if err != nil {
		fmt.Printf("Error searching for conversation: %s", err)
	}

	uniqIDs := make([]int, 0)
	unique := make(map[int]bool)
	for _, id := range ids {
		if !unique[id] {
			unique[id] = true
			uniqIDs = append(uniqIDs, id)
		}
	}

	fmt.Printf("Found %d conversations: ", len(uniqIDs))
	for i, id := range uniqIDs {
		fmt.Printf("%d", id)
		if i < len(uniqIDs)-1 {
			fmt.Printf(", ")
		}
	}
	fmt.Println()

	os.Exit(0)
}

func determineScreenSize() (int, int) {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return 80, 24
	}

	return width, height
}

func showConversation(convID int) {
	if viper.GetString("database.file") == "" {
		fmt.Println("Database file not set")
		os.Exit(1)
	}
	if viper.GetString("database.table") == "" {
		fmt.Println("Database table not set")
		os.Exit(1)
	}

	db, err := database.InitializeDB(os.ExpandEnv(viper.GetString("database.file")), viper.GetString("database.table"))
	if err != nil {
		fmt.Printf("Error opening database: %s", err)
		os.Exit(1)
	}
	defer db.Close()

	db.ShowConversation(convID)
	os.Exit(0)
}

func handleVersionFlags() bool {
	if viper.GetBool("version") {
		fmt.Println("ask-ollama version:", Version)
		return true
	}
	if viper.GetBool("full-version") {
		fmt.Printf("Version: %s\nCommit:  %s\nDate:    %s\n", Version, commit, date)
		return true
	}
	return false
}

func (c *Config) DumpConfig() {
	fmt.Printf("Config file: %s\n", viper.ConfigFileUsed())
	fmt.Printf("%s\n", c.String())
}

// // Example usage of roles
// func GetSystemPrompt(config *Config, roleName string) (string, error) {
// 	role, exists := config.Roles[roleName]
// 	if !exists {
// 		return "", fmt.Errorf("role '%s' not found in config", roleName)
// 	}
// 	return role.Prompt, nil
// }

// func main() {
// 	// Load configuration
// 	config, err := LoadConfig("config.yaml")
// 	if err != nil {
// 		log.Fatalf("Failed to load config: %v", err)
// 	}
//
// 	// Example: Get system prompt for developer role
// 	prompt, err := GetSystemPrompt(config, "developer")
// 	if err != nil {
// 		log.Fatalf("Error getting prompt: %v", err)
// 	}
//
// 	// Use the prompt in your LLM call
// 	fmt.Printf("Developer role prompt: %s\n", prompt)
//
// 	// Example: Access model configuration
// 	gpt4Config := config.Models["gpt4"]
// 	fmt.Printf("GPT-4 max tokens: %d\n", gpt4Config.MaxTokens)
// }
