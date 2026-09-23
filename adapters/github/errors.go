package github

import (
	"errors"
	"net/http"

	"zavictl/pkg/provider"

	githubapi "github.com/google/go-github/v64/github"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}

	var errorResp *githubapi.ErrorResponse
	if errors.As(err, &errorResp) {
		code := "GITHUB_API_ERROR"
		isRetryable := false
		
		switch errorResp.Response.StatusCode {
		case http.StatusUnauthorized:
			code = "GITHUB_UNAUTHORIZED"
		case http.StatusForbidden:
			code = "GITHUB_FORBIDDEN"
		case http.StatusNotFound:
			code = "GITHUB_NOT_FOUND"
		case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			code = "GITHUB_SERVER_ERROR"
			isRetryable = true
		}
		
		var rateLimitErr *githubapi.RateLimitError
		if errors.As(err, &rateLimitErr) {
			code = "GITHUB_RATE_LIMIT_EXCEEDED"
			isRetryable = true
		}
		
		var abuseErr *githubapi.AbuseRateLimitError
		if errors.As(err, &abuseErr) {
			code = "GITHUB_ABUSE_RATE_LIMIT"
			isRetryable = true
		}

		return &provider.PlatformError{
			ErrorCode:   code,
			Message:     errorResp.Message,
			IsRetryable: isRetryable,
		}
	}

	return &provider.PlatformError{
		ErrorCode:   "GITHUB_NETWORK_ERROR",
		Message:     err.Error(),
		IsRetryable: true,
	}
}
