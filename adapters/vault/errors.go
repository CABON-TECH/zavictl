package vault

import (
	"strings"

	"zavictl/pkg/provider"
	vaultapi "github.com/hashicorp/vault/api"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}

	code := "VAULT_API_ERROR"
	isRetryable := false
	msg := err.Error()

	if strings.Contains(msg, "permission denied") || strings.Contains(msg, "403") {
		code = "VAULT_FORBIDDEN"
	} else if strings.Contains(msg, "not found") || strings.Contains(msg, "404") {
		code = "VAULT_NOT_FOUND"
	} else if strings.Contains(msg, "connection refused") || strings.Contains(msg, "dial tcp") || strings.Contains(msg, "EOF") {
		code = "VAULT_NETWORK_ERROR"
		isRetryable = true
	} else if respErr, ok := err.(*vaultapi.ResponseError); ok {
		if respErr.StatusCode == 429 || respErr.StatusCode >= 500 {
			isRetryable = true
			if respErr.StatusCode == 429 {
				code = "VAULT_RATE_LIMIT"
			} else {
				code = "VAULT_SERVER_ERROR"
			}
		}
	}

	return &provider.PlatformError{
		ErrorCode:   code,
		Message:     msg,
		IsRetryable: isRetryable,
	}
}
