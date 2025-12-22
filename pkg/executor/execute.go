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

type Executor interface {
	Execute(cmd string, args ...string) ([]byte, error)
	ExecutorWtihTimeout(timeout time.Duration, cmd string, args ...string) ([]byte, error)
	ExecutorWithContext(ctx context.Context, cmd string, args ...string) ([]byte, error)
}

type executor struct{}

// Execute execute the named program with the given arguments,default timeout 20 minutes
func Execute(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	return ExecutorWithContext(ctx, name, args...)
}

// ExecutorWtihTimeout is like [Execute],can custom timeout
func ExecutorWtihTimeout(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return ExecutorWithContext(ctx, name, args...)
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
	return ExecutorWtihTimeout(timeout, "bash", "-c", cmd)
}

// ExecutorWithContext is like [Exeute] but includes a context.
// if the context is nil, it will be replaced with [context.WithTimeout] with a default timeout of 20 minutes.
func ExecutorWithContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "" {
		return nil, ErrEmptyCommand
	}

	cmd := exec.CommandContext(ctx, name, args...)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()

	if err == nil {
		return buf.Bytes(), nil
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return buf.Bytes(), fmt.Errorf("%w: %w", ErrTimeOut, err)
	case errors.Is(err, context.Canceled):
		return buf.Bytes(), fmt.Errorf("%w: %w", ErrCanceled, err)
	default:
		return buf.Bytes(), fmt.Errorf("%w: %w", ErrExit, err)
	}
}
