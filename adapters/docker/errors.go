package docker

import (
	"os/exec"
	"strings"

	"zavictl/pkg/provider"
)

func mapError(err error) error {
	return mapExecutionError(err, "")
}

func mapExecutionError(err error, stderr string) error {
	if err == nil {
		return nil
	}

	code := "DOCKER_EXEC_FAILED"
	isRetryable := false
	message := err.Error()

	if stderr != "" {
		message = stderr
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 0 {
			code = "DOCKER_CMD_ERROR"
			
			if strings.Contains(stderr, "timeout") || strings.Contains(stderr, "connection reset") {
				code = "DOCKER_NETWORK_ERROR"
				isRetryable = true
			} else if strings.Contains(stderr, "No such container") {
				code = "DOCKER_NOT_FOUND"
			}
		}
	}

	return &provider.PlatformError{
		ErrorCode:   code,
		Message:     message,
		IsRetryable: isRetryable,
	}
}
