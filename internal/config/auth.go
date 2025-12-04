package config

type Auth struct {
	Auth0Domain   string
	Auth0Audience string
}

func loadAuthConfig() (Auth, error) {
	auth0Domain, err := getEnv("AUTH0_DOMAIN")
	if err != nil {
		return Auth{}, err
	}

	auth0Audience, err := getEnv("AUTH0_AUDIENCE")
	if err != nil {
		return Auth{}, err
	}

	return Auth{
		Auth0Domain:   auth0Domain,
		Auth0Audience: auth0Audience,
	}, nil
}
