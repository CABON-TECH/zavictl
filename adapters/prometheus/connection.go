package prometheus

import (
	"context"
	"fmt"
	"time"

	"zavictl/pkg/provider"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type prometheusConnection struct {
	api promv1.API
}

func (c *prometheusConnection) Execute(ctx context.Context, op provider.Operation) (provider.Result, error) {
	switch op.Action {
	case "observability.query":
		return c.query(ctx, op)
	case "observability.query_range":
		return c.queryRange(ctx, op)
	default:
		return provider.Result{}, &provider.PlatformError{
			ErrorCode: "UNSUPPORTED_OPERATION",
			Message:   fmt.Sprintf("Prometheus provider does not support action: %s", op.Action),
		}
	}
}

func (c *prometheusConnection) Close() error {
	return nil
}

func (c *prometheusConnection) query(ctx context.Context, op provider.Operation) (provider.Result, error) {
	query, ok := op.Parameters["query"].(string)
	if !ok {
		return provider.Result{}, fmt.Errorf("missing parameter 'query'")
	}

	result, warnings, err := c.api.Query(ctx, query, time.Now())
	if err != nil {
		return provider.Result{}, mapError(err)
	}

	var parsedResults []any
	if result != nil {
		switch result.Type() {
		case model.ValVector:
			vec := result.(model.Vector)
			for _, sample := range vec {
				parsedResults = append(parsedResults, map[string]any{
					"metric": sample.Metric,
					"value":  float64(sample.Value),
					"time":   sample.Timestamp.Time().Unix(),
				})
			}
		case model.ValScalar:
			scalar := result.(*model.Scalar)
			parsedResults = append(parsedResults, map[string]any{
				"value": float64(scalar.Value),
				"time":  scalar.Timestamp.Time().Unix(),
			})
		}
	}
	
	resultType := ""
	if result != nil {
		resultType = result.Type().String()
	}

	return provider.Result{
		Status: "success",
		Outputs: map[string]any{
			"result_type": resultType,
			"results":     parsedResults,
			"warnings":    warnings,
		},
	}, nil
}

func (c *prometheusConnection) queryRange(ctx context.Context, op provider.Operation) (provider.Result, error) {
	query, ok := op.Parameters["query"].(string)
	if !ok {
		return provider.Result{}, fmt.Errorf("missing parameter 'query'")
	}

	result, warnings, err := c.api.QueryRange(ctx, query, promv1.Range{
		Start: time.Now().Add(-1 * time.Hour),
		End:   time.Now(),
		Step:  5 * time.Minute,
	})
	if err != nil {
		return provider.Result{}, mapError(err)
	}

	var parsedResults []any
	if result != nil && result.Type() == model.ValMatrix {
		matrix := result.(model.Matrix)
		for _, stream := range matrix {
			var values []map[string]any
			for _, val := range stream.Values {
				values = append(values, map[string]any{
					"value": float64(val.Value),
					"time":  val.Timestamp.Time().Unix(),
				})
			}
			parsedResults = append(parsedResults, map[string]any{
				"metric": stream.Metric,
				"values": values,
			})
		}
	}
	
	resultType := ""
	if result != nil {
		resultType = result.Type().String()
	}

	return provider.Result{
		Status: "success",
		Outputs: map[string]any{
			"result_type": resultType,
			"results":     parsedResults,
			"warnings":    warnings,
		},
	}, nil
}
