package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rousage/shortener/internal/appvalidator"
	"github.com/rousage/shortener/internal/auth"
	"github.com/rousage/shortener/internal/generator"
	"github.com/rousage/shortener/internal/repository"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type CreateShortUrlDTO struct {
	ShortCode string `json:"short_code" validate:"omitempty,min=5,max=16,shortcode"`
	URL       string `json:"url" validate:"required,http_url"`
}

func (s *Server) CreateShortURLHandler(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "CreateShortURLHandler")
	defer span.End()

	dto := new(CreateShortUrlDTO)

	if err := c.Bind(dto); err != nil {
		span.SetStatus(codes.Error, "failed to bind request")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(dto); err != nil {
		return s.failedValidationError(c, err)
	}
	span.SetAttributes(attribute.String("url", dto.URL))

	var (
		userId   = auth.GetUserID(c)
		rep      = repository.New(s.db)
		shortUrl string
		newUrl   repository.Url
		err      error
	)

	// Use a custom short code if provided,
	// otherwise generate a random one.
	// Only authenticated users can create custom short codes
	if dto.ShortCode != "" {
		span.SetAttributes(attribute.String("short_code", dto.ShortCode))

		if userId == nil || *userId == "" {
			span.AddEvent("unauthenticated user attempted to create custom short code")
			return echo.NewHTTPError(http.StatusForbidden, "Only authenticated users can create custom short codes")
		}

		newUrl, err = rep.CreateUrl(ctx, repository.CreateUrlParams{
			ID:       dto.ShortCode,
			LongUrl:  dto.URL,
			IsCustom: true,
			UserID:   userId,
		})
		if err != nil {
			span.SetStatus(codes.Error, "failed to create short url with custom short code")
			span.RecordError(err)

			if rep.IsDuplicateKeyError(err) {
				return c.JSON(http.StatusConflict, map[string]any{
					"message": "Validation failed",
					"errors": appvalidator.ValidationError{
						"short_code": "Short code is not available",
					},
				})
			}

			s.logger.Error().Err(err).Msg("failed to create custom short url")
			return echo.ErrInternalServerError
		}

		return c.JSON(http.StatusCreated, newUrl)
	}

	span.AddEvent("attempting to generate short url")
	const maxRetries = 3
	for attempt := range maxRetries {
		shortUrl, err = generator.ShortUrl(ctx, s.cfg.App.ShortUrlLength)
		if err != nil {
			break
		}

		newUrl, err = rep.CreateUrl(ctx, repository.CreateUrlParams{
			ID:       shortUrl,
			LongUrl:  dto.URL,
			IsCustom: false,
			UserID:   userId,
		})
		if err == nil {
			break
		}

		if rep.IsDuplicateKeyError(err) {
			span.AddEvent("Short URL collision detected, retrying", trace.WithAttributes(attribute.Int("attempt", attempt+1)))
			s.logger.Warn().Err(err).Int("attempt", attempt+1).Msg("Short URL collision detected, retrying")
			continue
		} else {
			break
		}
	}

	if err != nil {
		span.SetStatus(codes.Error, "failed to generate short url")
		span.RecordError(err)
		s.logger.Error().Err(err).Int("retries", maxRetries).Msg("failed to generate short url")
		return echo.ErrInternalServerError
	}

	span.AddEvent("short url generated")

	return c.JSON(http.StatusCreated, newUrl)
}

type GetLongUrlParams struct {
	Code string `param:"code" validate:"required"`
}

func (s *Server) GetLongUrlHandler(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "GetLongUrlHandler")
	defer span.End()

	params := new(GetLongUrlParams)
	if err := c.Bind(params); err != nil {
		span.SetStatus(codes.Error, "failed to bind request")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(params); err != nil {
		return s.failedValidationError(c, err)
	}
	span.SetAttributes(attribute.String("code", params.Code))

	longUrl, err := s.cache.GetLongUrl(ctx, params.Code)
	if err != nil {
		span.AddEvent("failed to get long url from cache")
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
		span.SetStatus(codes.Error, "failed to get long url")
		span.RecordError(err)

		if rep.IsNotFoundError(err) {
			s.logger.Error().Err(err).Str("code", params.Code).Msg("long url not found")
			return echo.ErrNotFound
		}

		s.logger.Error().Err(err).Str("code", params.Code).Msg("failed to get long url")
		return echo.ErrInternalServerError
	}

	if key, err := s.cache.SetLongUrl(ctx, params.Code, longUrl); err != nil {
		span.AddEvent("failed to cache long url", trace.WithAttributes(attribute.String("key", key)))
		s.logger.Warn().Err(err).Str("code", params.Code).Str("key", key).Msg("failed to cache long url ")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"longUrl": longUrl,
	})
}

type DeleteShortUrlParams struct {
	GetLongUrlParams
}

func (s *Server) DeletShortUrlHandler(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "DeleteShortUrlHandler")
	defer span.End()

	params := new(DeleteShortUrlParams)
	if err := c.Bind(params); err != nil {
		span.SetStatus(codes.Error, "failed to bind request")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(params); err != nil {
		return s.failedValidationError(c, err)
	}
	span.SetAttributes(attribute.String("code", params.Code))

	rep := repository.New(s.db)

	rowsAffected, err := rep.DeleteUrl(ctx, params.Code)
	if err != nil {
		span.SetStatus(codes.Error, "failed to delete short url")
		span.RecordError(err)

		s.logger.Error().Err(err).Str("code", params.Code).Msg("failed to delete short url")
		return echo.ErrInternalServerError
	}
	if rowsAffected == 0 {
		span.AddEvent("short url not found", trace.WithAttributes(attribute.String("code", params.Code)))
		s.logger.Warn().Str("code", params.Code).Msg("short url not found")
		return echo.ErrNotFound
	}

	if removedKeys, err := s.cache.DeleteLongUrl(ctx, params.Code); err != nil {
		span.AddEvent("failed to delete long url from cache", trace.WithAttributes(attribute.String("code", params.Code), attribute.Int64("removedKeys", removedKeys)))
		s.logger.Warn().Err(err).Str("code", params.Code).Str("code", params.Code).Int64("removedKeys", removedKeys).Msg("failed to delete long url from cache")
	}

	return c.NoContent(http.StatusNoContent)
}
