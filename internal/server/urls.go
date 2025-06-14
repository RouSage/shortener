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

func (s *Server) CreateShortURLHandler(c echo.Context) error {
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

	rep := repository.New(s.db)

	for attempt := range maxRetries {
		shortUrl, err = generator.ShortUrl(s.cfg.App.ShortUrlLength)
		if err != nil {
			break
		}

		newUrl, err = rep.CreateUrl(c.Request().Context(), repository.CreateUrlParams{
			ID:      shortUrl,
			LongUrl: dto.URL,
		})
		if err == nil {
			break
		}

		if rep.IsDuplicateKeyError(err) {
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

type GetLongUrlParams struct {
	Code string `param:"code" validate:"required"`
}

func (s *Server) getLongUrlHandler(c echo.Context) error {
	params := new(GetLongUrlParams)
	if err := c.Bind(params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(params); err != nil {
		return s.failedValidationError(c, err)
	}

	rep := repository.New(s.db)

	longUrl, err := rep.GetLongUrl(c.Request().Context(), params.Code)
	if err != nil {
		if rep.IsNotFoundError(err) {
			s.logger.Error().Err(err).Msgf("long url not found for code '%s'", params.Code)
			return echo.ErrNotFound
		}

		s.logger.Error().Err(err).Msgf("failed to get long url for code '%s'", params.Code)
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, map[string]string{
		"long_url": longUrl,
	})
}
