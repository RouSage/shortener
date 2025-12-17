package server

import (
	"net/http"
	"time"

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
	ShortCode string `json:"shortCode" validate:"omitempty,min=5,max=16,shortcode"`
	URL       string `json:"url" validate:"required,http_url"`
}

// createShortURLHandler godoc
//
//	@Summary		Create Short URL
//	@Description	Creates a shortened URL. Authenticated users can provide a custom short code (5-16 characters). Otherwise, a random code is generated.
//	@Tags			URLs
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateShortUrlDTO		true	"URL and optional custom short code"
//	@Success		201		{object}	repository.Url			"Created short URL"
//	@Failure		400		{object}	HTTPValidationError		"Validation failed"
//	@Failure		403		{object}	HTTPError				"Custom short codes require authentication"
//	@Failure		409		{object}	map[string]interface{}	"Short code already taken or validation failed"
//	@Failure		500		{object}	HTTPError				"Internal server error"
//	@Security		BearerAuth
//	@Router			/urls [post]
func (s *Server) createShortURLHandler(c echo.Context) error {
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
		span.SetAttributes(attribute.String("shortCode", dto.ShortCode))

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
						"shortCode": "Short code is not available",
					},
				})
			} else if rep.IsCheckConstraintError(err) {
				return c.JSON(http.StatusConflict, map[string]any{
					"message": "Validation failed",
					"errors": appvalidator.ValidationError{
						"shortCode": "Custom short code could not be created",
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
		} else if rep.IsCheckConstraintError(err) {
			return c.JSON(http.StatusConflict, map[string]any{
				"message": "Validation failed",
				"errors": appvalidator.ValidationError{
					"shortCode": "Custom short code could not be created",
				},
			})
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

// getLongUrlHandler godoc
//
//	@Summary		Get Long URL
//	@Description	Retrieves the original long URL for a given short code. Checks cache first, then database.
//	@Tags			URLs
//	@Produce		json
//	@Param			code	path		string				true	"Short code"
//	@Success		200		{object}	map[string]string	"longUrl"
//	@Failure		400		{object}	HTTPValidationError	"Validation failed"
//	@Failure		404		{object}	HTTPError			"Short URL not found"
//	@Failure		500		{object}	HTTPError			"Internal server error"
//	@Security		BearerAuth
//	@Router			/urls/{code} [get]
func (s *Server) getLongUrlHandler(c echo.Context) error {
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

type UrlResponse struct {
	ID        string    `json:"id"`
	LongUrl   string    `json:"longUrl"`
	CreatedAt time.Time `json:"createdAt"`
	IsCustom  bool      `json:"isCustom"`
}
type PaginatedUrls struct {
	Items      []UrlResponse `json:"items"`
	Pagination Pagination    `json:"pagination"`
}

// getUserUrls godoc
//
//	@Summary		Get User URLs
//	@Description	Retrieves a paginated list of URLs created by the authenticated user
//	@Tags			URLs
//	@Produce		json
//	@Param			page		query		int				    false	"Page number (min: 1, max: 10000)"	default(1)
//	@Param			pageSize	query		int				    false	"Page size (min: 1, max: 100)"		default(20)
//	@Success		200			{object}	PaginatedUrls	    "Paginated list of user URLs"
//	@Failure		400			{object}	HTTPValidationError	"Validation failed"
//	@Failure		401			{object}	HTTPError		    "Authentication required"
//	@Failure		500			{object}	HTTPError		    "Internal server error"
//	@Security		BearerAuth
//	@Router			/urls [get]
func (s *Server) getUserUrls(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "GetUserUrls")
	defer span.End()

	params := new(Filters)
	if err := c.Bind(params); err != nil {
		span.SetStatus(codes.Error, "failed to bind request")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(params); err != nil {
		return s.failedValidationError(c, err)
	}
	span.SetAttributes(attribute.Int("page", int(params.Page)), attribute.Int("pageSize", int(params.PageSize)))

	var (
		userID = auth.GetUserID(c)
		rep    = repository.New(s.db)
	)

	urls, err := rep.GetUserUrls(ctx, repository.GetUserUrlsParams{UserID: userID, Limit: params.limit(), Offset: params.offset()})
	if err != nil {
		span.SetStatus(codes.Error, "failed to get user urls")
		span.RecordError(err)

		return echo.ErrInternalServerError
	}

	var totalCount int
	if len(urls) > 0 {
		totalCount = int(urls[0].TotalCount)
	}

	items := make([]UrlResponse, len(urls))
	for i, url := range urls {
		items[i] = UrlResponse{
			ID:        url.ID,
			LongUrl:   url.LongUrl,
			CreatedAt: url.CreatedAt,
			IsCustom:  url.IsCustom,
		}
	}

	response := &PaginatedUrls{
		Items:      items,
		Pagination: calculatePagination(totalCount, int(params.Page), int(params.PageSize)),
	}

	return c.JSON(http.StatusOK, response)
}

type DeleteShortUrlParams struct {
	GetLongUrlParams
}

// deletShortUrlHandler godoc
//
//	@Summary		Delete Short URL
//	@Description	Deletes a short URL owned by the authenticated user. Also removes it from cache.
//	@Tags			URLs
//	@Produce		json
//	@Param			code	path	string	true	        "Short code to delete"
//	@Success		204		"No Content - URL successfully deleted"
//	@Failure		400		{object}	HTTPValidationError	"Validation failed"
//	@Failure		401		{object}	HTTPError	        "Authentication required"
//	@Failure		404		{object}	HTTPError	        "Short URL not found or not owned by user"
//	@Failure		500		{object}	HTTPError	        "Internal server error"
//	@Security		BearerAuth
//	@Router			/urls/{code} [delete]
func (s *Server) deletShortUrlHandler(c echo.Context) error {
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

	var (
		userID = auth.GetUserID(c)
		rep    = repository.New(s.db)
	)

	rowsAffected, err := rep.DeleteUrl(ctx, repository.DeleteUrlParams{ID: params.Code, UserID: userID})
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
