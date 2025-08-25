package utils

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/kirsle/configdir"
)

const (
	McpLogFileName = "mcp_logging.txt"
)

var (
	configDirPath = configdir.LocalConfig("apono-cli")
	mcpLogFile    *os.File
)

func InitMcpLogFile() error {
	logFilePath := path.Join(configDirPath, McpLogFileName)

	if mcpLogFile != nil {
		mcpLogFile.Close()
	}

	if err := configdir.MakePath(configDirPath); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to create MCP log file: %w", err)
	}

	mcpLogFile = file

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	header := fmt.Sprintf("=== Apono MCP Log Started at %s ===\n", timestamp)
	_, err = mcpLogFile.WriteString(header)
	if err != nil {
		return fmt.Errorf("failed to write header to log file: %w", err)
	}

	return nil
}

func McpLog(format string, args ...interface{}) {
	if mcpLogFile == nil {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	message := fmt.Sprintf(format, args...)
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)

	_, err := mcpLogFile.WriteString(logEntry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to write to MCP log file: %v\n", err)
	}
}

func CloseMcpLogFile() {
	if mcpLogFile != nil {
		mcpLogFile.Close()
		mcpLogFile = nil
	}
}
