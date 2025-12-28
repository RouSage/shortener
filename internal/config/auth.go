package config

type Auth struct {
	Auth0Domain       string
	Auth0Audience     string
	Auth0ClientID     string
	Auth0ClientSecret string
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

	auth0ClientID, err := getEnv("AUTH0_CLIENT_ID")
	if err != nil {
		return Auth{}, err
	}

	auth0ClientSecret, err := getEnv("AUTH0_CLIENT_SECRET")
	if err != nil {
		return Auth{}, err
	}

	return Auth{
		Auth0Domain:       auth0Domain,
		Auth0Audience:     auth0Audience,
		Auth0ClientID:     auth0ClientID,
		Auth0ClientSecret: auth0ClientSecret,
	}, nil
}
