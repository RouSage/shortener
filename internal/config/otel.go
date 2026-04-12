package config

import (
	"errors"
	"strconv"

	"log/slog"
)

type Otel struct {
	Enabled       bool
	Endpoint      string
	SamplingRatio float64
}

func loadOtelConfig(logger *slog.Logger) (Otel, error) {
	enabled, err := getBoolEnv("OTEL_ENABLED")
	if err != nil {
		return Otel{}, err
	}

	// Validate Grafana environment variables, they're required if OpenTelemetry is enabled
	// Do not add to the config, they're not needed in the code
	grafanaEndpoint := getOptionalEnv("GRAFANA_CLOUD_OTLP_ENDPOINT")
	if grafanaEndpoint == "" {
		logger.Warn("GRAFANA_CLOUD_OTLP_ENDPOINT is not specified")
		if enabled {
			return Otel{}, errors.New("GRAFANA_CLOUD_OTLP_ENDPOINT is required")
		}

	}
	grafanaInstanceID := getOptionalEnv("GRAFANA_CLOUD_INSTANCE_ID")
	if grafanaInstanceID == "" {
		logger.Warn("GRAFANA_CLOUD_INSTANCE_ID is not specified")
		if enabled {
			return Otel{}, errors.New("GRAFANA_CLOUD_INSTANCE_ID is required")
		}
	}
	grafanaAPIKey := getOptionalEnv("GRAFANA_CLOUD_API_KEY")
	if grafanaAPIKey == "" {
		logger.Warn("GRAFANA_CLOUD_API_KEY is not specified")
		if enabled {
			return Otel{}, errors.New("GRAFANA_CLOUD_API_KEY is required")
		}
	}
	// This expects a OpenTelemetry Collector gRPC endpoint
	endpoint := getOptionalEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		logger.Warn("OTEL_EXPORTER_OTLP_ENDPOINT is not specified")
		if enabled {
			return Otel{}, err
		}
	}

	// Get sampling ratio from environment, default to 1.0 (100%) if not set
	samplingRatio := 1.0
	if ratioStr := getOptionalEnv("OTEL_TRACES_SAMPLER_ARG"); ratioStr != "" {
		ratio, err := strconv.ParseFloat(ratioStr, 64)
		if err != nil {
			logger.Warn("invalid OTEL_TRACES_SAMPLER_ARG, using default 1.0", slog.String("value", ratioStr))
		} else if ratio < 0.0 || ratio > 1.0 {
			logger.Warn("OTEL_TRACES_SAMPLER_ARG must be between 0.0 and 1.0, using default 1.0", slog.Float64("value", ratio))
		} else {
			samplingRatio = ratio
		}
	}

	return Otel{
		Enabled:       enabled,
		Endpoint:      endpoint,
		SamplingRatio: samplingRatio,
	}, nil
}
