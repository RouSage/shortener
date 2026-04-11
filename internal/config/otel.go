package config

import (
	"strconv"

	"log/slog"
)

type Otel struct {
	TracesEndpoint string
	SamplingRatio  float64
}

func loadOtelConfig(logger *slog.Logger) (Otel, error) {
	// Validate Grafana environment variables
	// Do not add to the config, they're not needed in the code
	_, err := getEnv("GRAFANA_CLOUD_OTLP_ENDPOINT")
	if err != nil {
		return Otel{}, err
	}
	_, err = getEnv("GRAFANA_CLOUD_INSTANCE_ID")
	if err != nil {
		return Otel{}, err
	}
	_, err = getEnv("GRAFANA_CLOUD_API_KEY")
	if err != nil {
		return Otel{}, err
	}

	tracesEndpoint, err := getEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	if err != nil {
		return Otel{}, err
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
		TracesEndpoint: tracesEndpoint,
		SamplingRatio:  samplingRatio,
	}, nil
}
