package config

type Database struct {
	Username string
	Password string
	Host     string
	Port     int
	Database string
	Schema   string
}

func loadDatabaseConfig() (Database, error) {
	username, err := getEnv("DB_USERNAME")
	if err != nil {
		return Database{}, err
	}

	password, err := getEnv("DB_PASSWORD")
	if err != nil {
		return Database{}, err
	}

	host, err := getEnv("DB_HOST")
	if err != nil {
		return Database{}, err
	}

	port, err := getIntEnv("DB_PORT")
	if err != nil {
		return Database{}, err
	}

	database, err := getEnv("DB_DATABASE")
	if err != nil {
		return Database{}, err
	}

	schema, err := getEnv("DB_SCHEMA")
	if err != nil {
		return Database{}, err
	}

	return Database{
		Username: username,
		Password: password,
		Host:     host,
		Port:     port,
		Database: database,
		Schema:   schema,
	}, nil
}
