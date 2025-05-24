package config

type Server struct {
	Port int
}

func loadServerConfig() (Server, error) {
	port, err := getIntEnv("PORT")
	if err != nil {
		return Server{}, err
	}

	return Server{
		Port: port,
	}, nil
}
