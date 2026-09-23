package prometheus

import (
	"strings"

	"zavictl/pkg/provider"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}

	code := "PROMETHEUS_API_ERROR"
	isRetryable := false
	msg := err.Error()

	if apiErr, ok := err.(*promv1.Error); ok {
		switch apiErr.Type {
		case promv1.ErrBadData:
			code = "PROMETHEUS_BAD_QUERY"
		case promv1.ErrTimeout:
			code = "PROMETHEUS_TIMEOUT"
			isRetryable = true
		case promv1.ErrServer:
			code = "PROMETHEUS_SERVER_ERROR"
			isRetryable = true
		case promv1.ErrClient:
			code = "PROMETHEUS_CLIENT_ERROR"
		}
	} else if strings.Contains(msg, "connection refused") || strings.Contains(msg, "dial tcp") {
		code = "PROMETHEUS_NETWORK_ERROR"
		isRetryable = true
	}

	return &provider.PlatformError{
		ErrorCode:   code,
		Message:     msg,
		IsRetryable: isRetryable,
	}
}
