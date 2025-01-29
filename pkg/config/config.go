package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/duluk/ask-ollama/pkg/database"
)

type Options struct {
	Model          string
	Context        int
	ContextLength  int
	ContinueChat   bool
	DumpConfig     bool
	LogFileName    string
	DBFileName     string
	DBTable        string
	SystemPrompt   string
	MaxTokens      int
	Temperature    float32
	ConversationID int
	ScreenWidth    int
	ScreenHeight   int
	TabWidth       int
}

const Version = "0.3.3"

const MaxTermWidth = 80
const widthPad = 5
const TabWidth = 4

var (
	commit = "Unknown"
	date   = "Unknown"
)

func Initialize() (*Options, error) {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "ask-ollama")

	// TODO: though I've put so much effort into the config file to read it
	// first so that the values can be used as defaults (eg in --help), I'm
	// starting to wonder if that's even what I want...
	// Figure out the config file and read it first if there
	pflag.StringP("config", "C", "", "Configuration file")
	viper.BindPFlags(pflag.CommandLine)
	if err := setupConfigFile(); err != nil {
		return nil, fmt.Errorf("error setting up config: %w", err)
	}

	width, height := determineScreenSize()

	viper.SetDefault("model.default", "ollama")
	viper.SetDefault("model.context_length", 2048)
	viper.SetDefault("model.max_tokens", 512)
	viper.SetDefault("model.temperature", 0.7)
	viper.SetDefault("model.system_prompt", "")
	viper.SetDefault("log.file", filepath.Join(configDir, "ask-ollama.chat.yml"))
	viper.SetDefault("database.file", filepath.Join(configDir, "ask-ollama.db"))
	viper.SetDefault("database.table", "conversations")
	viper.SetDefault("screen.width", width)
	viper.SetDefault("screen.height", height)

	// Now define the rest of the flags using values from viper (which now has
	// config file values)
	pflag.StringP("model", "m", viper.GetString("model.default"), "Which LLM to use (claude|chatgpt|gemini|grok|deepseek|ollama)")
	pflag.IntP("context", "n", 0, "Use previous n messages for context")
	pflag.IntP("context-length", "l", viper.GetInt("model.context_length"), "Maximum context length")
	pflag.BoolP("continue", "c", false, "Continue previous conversation")
	pflag.StringP("log", "L", viper.GetString("log.file"), "Chat log file")
	pflag.StringP("database", "d", viper.GetString("database.file"), "Database file")
	pflag.StringP("system-prompt", "S", viper.GetString("model.system_prompt"), "System prompt for LLM")
	pflag.IntP("max-tokens", "t", viper.GetInt("model.max_tokens"), "Maximum tokens to generate")
	pflag.Float32P("temperature", "T", float32(viper.GetFloat64("model.temperature")), "Temperature for generation")
	pflag.BoolP("version", "v", false, "Print version and exit")
	pflag.BoolP("full-version", "V", false, "Print full version information and exit")
	pflag.BoolP("dump-config", "", false, "Dump configuration and exit")
	pflag.IntP("id", "i", 0, "ID of the conversation to continue")
	pflag.StringP("search", "s", "", "Search for a conversation")
	pflag.IntP("show", "", 0, "Show conversation with ID")
	pflag.StringP("width", "", "", "Width of the screen for linewrap")
	pflag.StringP("height", "", "", "Height of the screen for linewrap")

	// Bind all flags to viper
	viper.BindPFlag("context", pflag.Lookup("context"))
	viper.BindPFlag("continue", pflag.Lookup("continue"))
	viper.BindPFlag("dump-config", pflag.Lookup("dump-config"))
	viper.BindPFlag("id", pflag.Lookup("id"))
	viper.BindPFlag("search", pflag.Lookup("search"))
	viper.BindPFlag("show", pflag.Lookup("show"))
	viper.BindPFlag("log.file", pflag.Lookup("log"))
	viper.BindPFlag("database.file", pflag.Lookup("database"))
	viper.BindPFlag("model.system_prompt", pflag.Lookup("system-prompt"))
	viper.BindPFlag("model.max_tokens", pflag.Lookup("max-tokens"))
	viper.BindPFlag("model.context_length", pflag.Lookup("context-length"))
	viper.BindPFlag("model.temperature", pflag.Lookup("temperature"))
	viper.BindPFlag("screen.width", pflag.Lookup("width"))
	viper.BindPFlag("screen.height", pflag.Lookup("height"))

	viper.BindPFlag("version", pflag.Lookup("version"))
	viper.BindPFlag("full-version", pflag.Lookup("full-version"))

	pflag.Parse()

	// Handle version flags and bail if necessary
	if handleVersionFlags() {
		os.Exit(0)
	}

	if viper.GetString("search") != "" {
		searchForConversation(viper.GetString("search"))
	}

	if viper.GetInt("show") != 0 {
		showConversation(viper.GetInt("show"))
	}

	// Create and return options
	return &Options{
		Model:          pflag.Lookup("model").Value.String(),
		Context:        viper.GetInt("context"),
		ContextLength:  viper.GetInt("model.context_length"),
		ContinueChat:   viper.GetBool("continue"),
		DumpConfig:     viper.GetBool("dump-config"),
		LogFileName:    os.ExpandEnv(viper.GetString("log.file")),
		DBFileName:     os.ExpandEnv(viper.GetString("database.file")),
		DBTable:        viper.GetString("database.table"),
		SystemPrompt:   viper.GetString("model.system_prompt"),
		MaxTokens:      viper.GetInt("model.max_tokens"),
		Temperature:    float32(viper.GetFloat64("model.temperature")),
		ConversationID: viper.GetInt("id"),
		ScreenWidth:    min(viper.GetInt("screen.width"), MaxTermWidth) - widthPad,
		ScreenHeight:   viper.GetInt("screen.height"),
		TabWidth:       TabWidth,
	}, nil
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

	db.ShowConversation(viper.GetInt("show"))
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

// This is pretty bad but it's a quick hack to get the config file because
// nothing else is working. I need to parse and read the config file before
// the rest of the options so that I can use the values from the config file
// to set the defaults for the other options.
func checkConfigFlag() string {
	for i, arg := range os.Args {
		if arg == "--config" || arg == "-C" {
			if i+1 < len(os.Args) {
				return os.Args[i+1]
			}
		}
		if len(arg) > 8 && arg[:8] == "--config=" {
			return arg[8:]
		}
	}
	return ""
}

// If there's an error getting the user, just returning the path unmodified
func expandHomePath(path string) string {
	if strings.HasPrefix(path, "~") {
		currentUser, err := user.Current()
		if err != nil {
			// I'm always trying to sneak a goto in just to trigger :)
			goto oopsies
		}
		return filepath.Join(currentUser.HomeDir, path[1:])
	}
oopsies:
	return path
}

func setupConfigFile() error {
	cfgFile := checkConfigFlag()

	if cfgFile != "" {
		viper.SetConfigFile(expandHomePath(cfgFile))
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yml")
		viper.AddConfigPath(filepath.Join(os.Getenv("HOME"), ".config", "ask-ollama"))
		viper.AddConfigPath(".")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}
	return nil
}

func DumpConfig(cfg *Options) {
	fmt.Printf("Model: %s\n", cfg.Model)
	fmt.Printf("Context: %d\n", cfg.Context)
	fmt.Printf("ContextLength: %d\n", cfg.ContextLength)
	fmt.Printf("ContinueChat: %t\n", cfg.ContinueChat)
	fmt.Printf("LogFileName: %s\n", cfg.LogFileName)
	fmt.Printf("DBFileName: %s\n", cfg.DBFileName)
	fmt.Printf("DBTable: %s\n", cfg.DBTable)
	fmt.Printf("SystemPrompt: %s\n", cfg.SystemPrompt)
	fmt.Printf("MaxTokens: %d\n", cfg.MaxTokens)
	fmt.Printf("Temperature: %f\n", cfg.Temperature)
	fmt.Printf("ConversationID: %d\n", cfg.ConversationID)
	fmt.Printf("ScreenWidth: %d\n", cfg.ScreenWidth)
	fmt.Printf("ScreenHeight: %d\n", cfg.ScreenHeight)
	fmt.Printf("TabWidth: %d\n", cfg.TabWidth)
}
