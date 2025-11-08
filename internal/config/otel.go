package config

type Otel struct {
	TracesEndpoint string
}

func loadOtelConfig() (Otel, error) {
	env, err := getEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	if err != nil {
		return Otel{}, err
	}

	return Otel{
		TracesEndpoint: env,
	}, nil
}
