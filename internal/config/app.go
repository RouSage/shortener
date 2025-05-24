package config

type App struct {
	Env string
}

func loadAppConfig() (App, error) {
	env, err := getEnv("APP_ENV")
	if err != nil {
		return App{}, err
	}

	return App{Env: env}, nil
}
