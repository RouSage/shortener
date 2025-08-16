package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rousage/shortener/internal/cache"
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
			ID:       shortUrl,
			LongUrl:  dto.URL,
			IsCustom: false,
		})
		if err == nil {
			break
		}

		if rep.IsDuplicateKeyError(err) {
			s.logger.Warn().Err(err).Int("attempt", attempt+1).Msg("Short URL collision detected, retrying")
			continue
		} else {
			break
		}
	}

	if err != nil {
		s.logger.Error().Err(err).Int("retries", maxRetries).Msg("failed to generate short url")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusCreated, newUrl)
}

type GetLongUrlParams struct {
	Code string `param:"code" validate:"required"`
}

func (s *Server) GetLongUrlHandler(c echo.Context) error {
	params := new(GetLongUrlParams)
	if err := c.Bind(params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(params); err != nil {
		return s.failedValidationError(c, err)
	}

	ctx := c.Request().Context()
	cache := cache.New(s.cache)

	longUrl, err := cache.GetLongUrl(ctx, params.Code)
	if err != nil {
		s.logger.Warn().Err(err).Str("code", params.Code).Msg("failed to get long url from cache")
	}
	if longUrl != "" {
		return c.JSON(http.StatusOK, map[string]string{
			"longUrl": longUrl,
		})
	}

	rep := repository.New(s.db)
	longUrl, err = rep.GetLongUrl(ctx, params.Code)
	if err != nil {
		if rep.IsNotFoundError(err) {
			s.logger.Error().Err(err).Str("code", params.Code).Msg("long url not found")
			return echo.ErrNotFound
		}

		s.logger.Error().Err(err).Str("code", params.Code).Msg("failed to get long url")
		return echo.ErrInternalServerError
	}

	if key, err := cache.SetLongUrl(ctx, params.Code, longUrl); err != nil {
		s.logger.Warn().Err(err).Str("code", params.Code).Str("key", key).Msg("failed to cache long url ")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"longUrl": longUrl,
	})
}
