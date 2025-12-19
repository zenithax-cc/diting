package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const Timeout = 20 * time.Minute

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
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	return ExecutorWithContext(ctx, name, args...)
}

// ExecutorWtihTimeout is like [Execute],can custom timeout
func ExecutorWtihTimeout(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return ExecutorWithContext(ctx, name, args...)
}

// ExecutorWithContext is like [Exeute] but includes a context.
// if the context is nil, it will be replaced with [context.WithTimeout] with a default timeout of 20 minutes.
func ExecutorWithContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "" {
		return nil, ErrEmptyCommand
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), Timeout)
		defer cancel()
	}

	argString := strings.Join(args, " ")
	cmds := []string{name, argString}

	cmd := exec.CommandContext(ctx, "bash -c ", cmds...)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()

	if err == nil {
		return buf.Bytes(), nil
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		err = ErrTimeOut
	case errors.Is(err, context.Canceled):
		err = ErrCanceled
	default:
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			err = ErrExit
		}
	}

	return buf.Bytes(), fmt.Errorf("%w : %v", err, cmds)
}

