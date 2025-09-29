package util

import (
	"errors"
	"fmt"
)

// ContainerError represents errors specific to container operations
type ContainerError struct {
	Op   string // Operation that failed
	Path string // Path related to the error (optional)
	Err  error  // Underlying error
}

func (e *ContainerError) Error() string {
	if e.Err == nil {
		// Handle case where no underlying error is provided
		if e.Path != "" {
			return fmt.Sprintf("%s %s", e.Op, e.Path)
		}
		return e.Op
	}

	if e.Path != "" {
		return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *ContainerError) Unwrap() error {
	return e.Err
}

// NewError creates a new ContainerError
func NewError(op string, err error) error {
	return &ContainerError{Op: op, Err: err}
}

// NewSimpleError creates a new ContainerError with a descriptive message
func NewSimpleError(op string, message string) error {
	return &ContainerError{Op: op, Err: errors.New(message)}
}

// NewPathError creates a new ContainerError with a path
func NewPathError(op string, path string, err error) error {
	return &ContainerError{Op: op, Path: path, Err: err}
}

// WrapError wraps an existing error with additional context
func WrapError(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}