package config

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/apono-io/apono-cli/pkg/commands/mcp-proxy/auditor"
	"gopkg.in/yaml.v3"
)

// Config represents the complete MCP proxy configuration
type Config struct {
	Proxy ProxyConfig `yaml:"proxy"`
	Risk  RiskConfig  `yaml:"risk"`
	Slack SlackConfig `yaml:"slack"`
}

// ProxyConfig represents the proxy endpoint configuration
type ProxyConfig struct {
	Name     string            `yaml:"name"`
	Endpoint string            `yaml:"endpoint"`
	Headers  map[string]string `yaml:"headers"`
	Command  string            `yaml:"command"`
	Args     []string          `yaml:"args"`
	Env      map[string]string `yaml:"env"`
}

// RiskConfig represents risk detection configuration
type RiskConfig struct {
	Enabled        bool     `yaml:"enabled"`
	BlockOnRisk    bool     `yaml:"block_on_risk"`
	RiskyMethods   []string `yaml:"risky_methods"`
	RiskyKeywords  []string `yaml:"risky_keywords"`
	AllowedMethods []string `yaml:"allowed_methods"`
}

// SlackConfig represents Slack integration configuration
type SlackConfig struct {
	Enabled          bool          `yaml:"enabled"`
	BotToken         string        `yaml:"bot_token"`
	ChannelID        string        `yaml:"channel_id"` // Optional: Channel ID (starts with C) or User ID (starts with U) for DMs
	UserID           string        `yaml:"user_id"`    // Optional: User ID for direct messages (takes precedence over channel_id)
	SigningSecret    string        `yaml:"signing_secret"`
	SkipVerification bool          `yaml:"skip_verification"` // Optional: Skip signature verification (useful for debugging)
	CallbackPort     int           `yaml:"callback_port"`
	Timeout          time.Duration `yaml:"timeout"`
}

// LoadConfig loads configuration from a YAML file with environment variable substitution
func LoadConfig(path string) (*Config, error) {
	// Read the config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Substitute environment variables
	expandedData := expandEnvVars(string(data))

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal([]byte(expandedData), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// expandEnvVars replaces ${VAR} or $VAR patterns with environment variable values
func expandEnvVars(input string) string {
	// Match ${VAR} pattern
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	result := re.ReplaceAllStringFunc(input, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match // Keep original if not found
	})

	// Match $VAR pattern (without braces)
	re2 := regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
	result = re2.ReplaceAllStringFunc(result, func(match string) string {
		varName := match[1:] // Remove $
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match // Keep original if not found
	})

	return result
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate Slack configuration if enabled
	if config.Slack.Enabled {
		if config.Slack.BotToken == "" {
			return fmt.Errorf("slack bot_token is required when slack is enabled")
		}
		// Either channel_id or user_id must be specified
		if config.Slack.ChannelID == "" && config.Slack.UserID == "" {
			return fmt.Errorf("either slack channel_id or user_id is required when slack is enabled")
		}
		// Signing secret is required unless verification is explicitly skipped
		if config.Slack.SigningSecret == "" && !config.Slack.SkipVerification {
			return fmt.Errorf("slack signing_secret is required when slack is enabled (or set skip_verification: true)")
		}
		fmt.Println("slack callback_port", config.Slack.CallbackPort)
		if config.Slack.CallbackPort <= 0 {
			return fmt.Errorf("slack callback_port must be a positive number")
		}
		if config.Slack.Timeout <= 0 {
			config.Slack.Timeout = 5 * time.Minute // Default timeout
		}
	}

	return nil
}

// ToRiskConfig converts Config.Risk to auditor.RiskConfig
func (c *Config) ToRiskConfig() auditor.RiskConfig {
	return auditor.RiskConfig{
		Enabled:        c.Risk.Enabled,
		BlockOnRisk:    c.Risk.BlockOnRisk,
		RiskyMethods:   c.Risk.RiskyMethods,
		RiskyKeywords:  c.Risk.RiskyKeywords,
		AllowedMethods: c.Risk.AllowedMethods,
	}
}
