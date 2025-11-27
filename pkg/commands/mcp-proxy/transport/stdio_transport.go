package transport

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/apono-io/apono-cli/pkg/utils"
)

// STDIOProxyConfig configures the STDIO proxy for subprocess mode
type STDIOProxyConfig struct {
	Command string
	Args    []string
	Env     map[string]string
	Debug   bool
}

// RunSTDIOProxy runs a simple STDIO<->STDIO MCP proxy with subprocess
func RunSTDIOProxy(config STDIOProxyConfig) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	utils.McpLogf("=== STDIO Proxy Started ===")
	utils.McpLogf("Spawning subprocess: %s %v", config.Command, config.Args)

	// Create the command
	cmd := exec.CommandContext(ctx, config.Command, config.Args...)

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Connect pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the subprocess
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start subprocess: %w", err)
	}
	utils.McpLogf("Subprocess started successfully")

	// Handle stderr in background
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			utils.McpLogf("[Subprocess STDERR]: %s", scanner.Text())
		}
	}()

	// Set up bidirectional communication
	errCh := make(chan error, 2)
	doneCh := make(chan struct{})

	// Forward stdin to subprocess
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			if config.Debug {
				utils.McpLogf("[Client→Subprocess]: %s", line)
			}

			if _, err := fmt.Fprintln(stdin, line); err != nil {
				utils.McpLogf("[Error]: Failed to write to subprocess: %v", err)
				errCh <- err
				return
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("error reading stdin: %w", err)
			return
		}
		// Client closed stdin, close subprocess stdin
		stdin.Close()
		doneCh <- struct{}{}
	}()

	// Forward subprocess stdout to our stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if config.Debug {
				utils.McpLogf("[Subprocess→Client]: %s", line)
			}

			fmt.Println(line)
		}
		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("error reading subprocess stdout: %w", err)
			return
		}
		doneCh <- struct{}{}
	}()

	// Wait for completion
	select {
	case <-ctx.Done():
		utils.McpLogf("Context cancelled, shutting down...")
		stdin.Close()
		cmd.Wait()
		return nil
	case err := <-errCh:
		utils.McpLogf("Error in proxy: %v", err)
		stdin.Close()
		cmd.Wait()
		return err
	case <-doneCh:
		utils.McpLogf("Communication channel closed, waiting for subprocess...")
		stdin.Close()
		if err := cmd.Wait(); err != nil {
			utils.McpLogf("Subprocess exited with error: %v", err)
			return err
		}
		utils.McpLogf("Subprocess completed successfully")
		return nil
	}
}
