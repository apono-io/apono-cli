package utils

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/kirsle/configdir"

	"github.com/apono-io/apono-cli/pkg/config"
)

const (
	McpLogFileName = "mcp_logging.log"
)

var mcpLogFile *os.File

func InitMcpLogFile() error {
	logFilePath := path.Join(config.DirPath, McpLogFileName)

	if mcpLogFile != nil {
		err := mcpLogFile.Close()
		if err != nil {
			return err
		}
	}

	if err := configdir.MakePath(config.DirPath); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.Create(filepath.Clean(logFilePath))
	if err != nil {
		return fmt.Errorf("failed to create MCP Server log file: %w", err)
	}

	mcpLogFile = file

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	header := fmt.Sprintf("=== Apono MCP Server Log Started at %s ===\n", timestamp)
	_, err = mcpLogFile.WriteString(header)
	if err != nil {
		return fmt.Errorf("failed to write header to log file: %w", err)
	}

	return nil
}

func McpLogf(format string, args ...interface{}) {
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
		err := mcpLogFile.Close()
		if err != nil {
			return
		}
		mcpLogFile = nil
	}
}
