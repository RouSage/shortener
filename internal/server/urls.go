package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rousage/shortener/internal/generator"
	"github.com/rousage/shortener/internal/repository"
)

type CreateShortUrlDTO struct {
	URL string `json:"url" validate:"required,http_url"`
}

func (s *Server) createShortURLHandler(c echo.Context) error {
	dto := new(CreateShortUrlDTO)

	if err := c.Bind(dto); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(dto); err != nil {
		return s.failedValidationError(c, err)
	}

	const maxRetries = 3
	var (
		shortUrl string
		newUrl   repository.Url
		err      error
	)

	r := repository.New(s.db)

	for attempt := range maxRetries {
		shortUrl, err = generator.ShortUrl(s.cfg.App.ShortUrlLength)
		if err != nil {
			break
		}

		newUrl, err = r.CreateUrl(c.Request().Context(), repository.CreateUrlParams{
			ID:      shortUrl,
			LongUrl: dto.URL,
		})
		if err == nil {
			break
		}

		if r.IsDuplicateKeyError(err) {
			s.logger.Warn().Err(err).Msgf("Short URL collision detected on attempt %d, retrying", attempt+1)
			continue
		} else {
			break
		}
	}

	if err != nil {
		s.logger.Error().Err(err).Msgf("failed to generate short url after %d attempts", maxRetries)
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusCreated, newUrl)
}
