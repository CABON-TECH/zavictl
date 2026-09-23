package opentofu

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

	code := "TOFU_EXEC_FAILED"
	isRetryable := false
	message := err.Error()

	if stderr != "" {
		message = stderr
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 1 {
			code = "TOFU_CONFIG_ERROR"
			
			if strings.Contains(stderr, "timeout") || strings.Contains(stderr, "connection reset") {
				code = "TOFU_NETWORK_ERROR"
				isRetryable = true
			} else if strings.Contains(stderr, "state lock") {
				code = "TOFU_STATE_LOCKED"
				isRetryable = true
			}
		}
	}

	return &provider.PlatformError{
		ErrorCode:   code,
		Message:     message,
		IsRetryable: isRetryable,
	}
}
