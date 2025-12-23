package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

const DefaultTimeout = 20 * time.Minute

var (
	ErrEmptyCommand = errors.New("empty command")
	ErrTimeOut      = errors.New("command timed out")
	ErrCanceled     = errors.New("command canceled")
	ErrExit         = errors.New("command exited with error")
)

// Execute execute the named program with the given arguments,default timeout 20 minutes
func Execute(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	return ExecuteWithContext(ctx, name, args...)
}

// ExecuteWtihTimeout is like [Execute],can custom timeout
func ExecuteWtihTimeout(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return ExecuteWithContext(ctx, name, args...)
}

func ExecuteShell(cmd string) ([]byte, error) {
	if cmd == "" {
		return nil, ErrEmptyCommand
	}
	return Execute("bash", "-c", cmd)
}

func ExecuteShellWithTimeout(timeout time.Duration, cmd string) ([]byte, error) {
	if cmd == "" {
		return nil, ErrEmptyCommand
	}
	return ExecuteWtihTimeout(timeout, "bash", "-c", cmd)
}

// ExecutorWithContext is like [Exeute] but includes a context.
// if the context is nil, it will be replaced with [context.WithTimeout] with a default timeout of 20 minutes.
func ExecuteWithContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "" {
		return nil, ErrEmptyCommand
	}

	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	cmd := exec.CommandContext(ctx, name, args...)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()

	var exitErr error
	if err != nil {
		switch {
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			exitErr = fmt.Errorf("%s: %w", ErrTimeOut, err)
		case errors.Is(ctx.Err(), context.Canceled):
			exitErr = fmt.Errorf("%s: %w", ErrCanceled, err)
		default:
			exitErr = fmt.Errorf("%s: %w", ErrExit, err)
		}
	}

	return buf.Bytes(), exitErr
}
