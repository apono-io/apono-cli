package config

import (
	"os"
	"testing"
	"time"
)

func TestExpandEnvVars(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_VAR", "test_value")
	os.Setenv("SLACK_TOKEN", "xoxb-test-token")
	defer os.Unsetenv("TEST_VAR")
	defer os.Unsetenv("SLACK_TOKEN")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no variables",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "braced variable",
			input:    "token: ${SLACK_TOKEN}",
			expected: "token: xoxb-test-token",
		},
		{
			name:     "unbraced variable",
			input:    "value: $TEST_VAR",
			expected: "value: test_value",
		},
		{
			name:     "multiple variables",
			input:    "${TEST_VAR} and ${SLACK_TOKEN}",
			expected: "test_value and xoxb-test-token",
		},
		{
			name:     "non-existent variable",
			input:    "${NON_EXISTENT}",
			expected: "${NON_EXISTENT}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config with slack disabled",
			config: &Config{
				Slack: SlackConfig{
					Enabled: false,
				},
			},
			expectError: false,
		},
		{
			name: "valid config with slack enabled",
			config: &Config{
				Slack: SlackConfig{
					Enabled:       true,
					BotToken:      "xoxb-test",
					ChannelID:     "C123",
					SigningSecret: "secret",
					CallbackPort:  8011,
					Timeout:       5 * time.Minute,
				},
			},
			expectError: false,
		},
		{
			name: "missing bot token",
			config: &Config{
				Slack: SlackConfig{
					Enabled:       true,
					ChannelID:     "C123",
					SigningSecret: "secret",
					CallbackPort:  8011,
				},
			},
			expectError: true,
		},
		{
			name: "missing channel id",
			config: &Config{
				Slack: SlackConfig{
					Enabled:       true,
					BotToken:      "xoxb-test",
					SigningSecret: "secret",
					CallbackPort:  8011,
				},
			},
			expectError: true,
		},
		{
			name: "missing signing secret",
			config: &Config{
				Slack: SlackConfig{
					Enabled:      true,
					BotToken:     "xoxb-test",
					ChannelID:    "C123",
					CallbackPort: 8011,
				},
			},
			expectError: true,
		},
		{
			name: "invalid callback port",
			config: &Config{
				Slack: SlackConfig{
					Enabled:       true,
					BotToken:      "xoxb-test",
					ChannelID:     "C123",
					SigningSecret: "secret",
					CallbackPort:  0,
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestToRiskConfig(t *testing.T) {
	cfg := &Config{
		Risk: RiskConfig{
			Enabled:      true,
			BlockOnRisk:  false,
			RiskyMethods: []string{"delete", "drop"},
		},
	}

	riskCfg := cfg.ToRiskConfig()

	if !riskCfg.Enabled {
		t.Error("Expected risk config to be enabled")
	}

	if riskCfg.BlockOnRisk {
		t.Error("Expected BlockOnRisk to be false")
	}

	if len(riskCfg.RiskyMethods) != 2 {
		t.Errorf("Expected 2 risky methods, got %d", len(riskCfg.RiskyMethods))
	}
}
