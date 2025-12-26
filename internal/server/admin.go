package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rousage/shortener/internal/repository"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type URLsFilters struct {
	PaginationFilters
	IsCustom *bool   `query:"isCustom" validate:"omitzero,boolean"`
	UserID   *string `query:"userId" validate:"omitzero,min=1,max=50"`
}
type PaginatedURLs struct {
	Items      []repository.Url `json:"items"`
	Pagination Pagination       `json:"pagination"`
}

// getURLs godoc
//
//	@Summary		Get all URLs
//	@Description	Retrieves a paginated list of all URLs created by users
//	@Tags			URLs,Admin
//	@Produce		json
//	@Param			isCustom	query		bool				false	"Get custom URLs only"
//	@Param			userId		query		string				false	"Get URLs created by a specific user"	minlength(1)	maxlength(50)
//	@Param			page		query		int					true	"Page number"							minimum(1)		maximum(10000)	default(1)
//	@Param			pageSize	query		int					true	"Page size"								minimum(1)		maximum(100)	default(20)
//	@Success		200			{object}	PaginatedURLs		"Paginated list of URLs"
//	@Failure		400			{object}	HTTPValidationError	"Validation failed"
//	@Failure		401			{object}	HTTPError			"Unauthorized"
//	@Failure		403			{object}	HTTPError			"Forbidden"
//	@Failure		500			{object}	HTTPError			"Internal server error"
//	@Security		BearerAuth
//	@Router			/v1/admin/urls [get]
func (s *Server) getURLs(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "admin.GetURLs")
	defer span.End()

	params := new(URLsFilters)
	if err := c.Bind(params); err != nil {
		span.SetStatus(codes.Error, "failed to bind request")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(params); err != nil {
		return s.failedValidationError(c, err)
	}

	span.SetAttributes(attribute.Int("page", int(params.Page)), attribute.Int("pageSize", int(params.PageSize)))
	if params.IsCustom != nil {
		span.SetAttributes(attribute.Bool("isCustom", *params.IsCustom))
	}
	if params.UserID != nil {
		span.SetAttributes(attribute.String("userId", *params.UserID))
	}

	var rep = repository.New(s.db)
	urls, err := rep.GetURLs(ctx, repository.GetURLsParams{IsCustom: params.IsCustom, UserID: params.UserID, Limit: params.limit(), Offset: params.offset()})
	if err != nil {
		span.SetStatus(codes.Error, "failed to get urls")
		span.RecordError(err)

		return echo.ErrInternalServerError
	}

	var totalCount int
	if len(urls) > 0 {
		totalCount = int(urls[0].TotalCount)
	}

	items := make([]repository.Url, len(urls))
	for i, url := range urls {
		items[i] = repository.Url{
			ID:        url.ID,
			LongUrl:   url.LongUrl,
			CreatedAt: url.CreatedAt,
			IsCustom:  url.IsCustom,
			UserID:    url.UserID,
		}
	}

	response := &PaginatedURLs{
		Items:      items,
		Pagination: calculatePagination(totalCount, int(params.Page), int(params.PageSize)),
	}

	return c.JSON(http.StatusOK, response)
}
