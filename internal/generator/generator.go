package generator

import gonanoid "github.com/matoous/go-nanoid/v2"

const defaultLength = 8

func ShortUrl(length int) (string, error) {
	if length <= 0 {
		length = defaultLength
	}

	id, err := gonanoid.New(length)
	if err != nil {
		return "", err
	}

	return id, nil
}
