package config

type Cache struct {
	Host string
	Port int
}

func loadCacheConfig() (Cache, error) {
	host, err := getEnv("VALKEY_HOST")
	if err != nil {
		return Cache{}, err
	}

	port, err := getIntEnv("VALKEY_PORT")
	if err != nil {
		return Cache{}, err
	}

	return Cache{
		Host: host,
		Port: port,
	}, nil
}
