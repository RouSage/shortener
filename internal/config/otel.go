package config

import (
	"strconv"

	"github.com/rs/zerolog/log"
)

type Otel struct {
	TracesEndpoint string
	SamplingRatio  float64
}

func loadOtelConfig() (Otel, error) {
	tracesEndpoint, err := getEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	if err != nil {
		return Otel{}, err
	}

	// Get sampling ratio from environment, default to 1.0 (100%) if not set
	samplingRatio := 1.0
	if ratioStr := getOptionalEnv("OTEL_TRACES_SAMPLER_ARG"); ratioStr != "" {
		ratio, err := strconv.ParseFloat(ratioStr, 64)
		if err != nil {
			log.Warn().Str("value", ratioStr).Msg("invalid OTEL_TRACES_SAMPLER_ARG, using default 1.0")
		} else if ratio < 0.0 || ratio > 1.0 {
			log.Warn().Float64("value", ratio).Msg("OTEL_TRACES_SAMPLER_ARG must be between 0.0 and 1.0, using default 1.0")
		} else {
			samplingRatio = ratio
		}
	}

	return Otel{
		TracesEndpoint: tracesEndpoint,
		SamplingRatio:  samplingRatio,
	}, nil
}
