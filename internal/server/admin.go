package server

import (
	"errors"
	"net/http"

	"github.com/auth0/go-auth0/v2/management/core"
	"github.com/labstack/echo/v4"
	"github.com/rousage/shortener/internal/auth"
	"github.com/rousage/shortener/internal/repository"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
//	@Tags			Admin
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

	urls, err := s.rep.GetURLs(ctx, repository.GetURLsParams{IsCustom: params.IsCustom, UserID: params.UserID, Limit: params.limit(), Offset: params.offset()})
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

type DeleteURLParams struct {
	GetLongUrlParams
}

// deleteURLHandler godoc
//
//	@Summary		Delete URL
//	@Description	Deletes a URL. Also removes it from cache.
//	@Tags			Admin
//	@Produce		json
//	@Param			code	path	string	true	"Short code of the URL"	maxlength(16)
//	@Success		204		"No Content - URL successfully deleted"
//	@Failure		400		{object}	HTTPValidationError	"Validation failed"
//	@Failure		401		{object}	HTTPError			"Unauthorized"
//	@Failure		403		{object}	HTTPError			"Forbidden"
//	@Failure		404		{object}	HTTPError			"Short URL not found"
//	@Failure		500		{object}	HTTPError			"Internal server error"
//	@Security		BearerAuth
//	@Router			/v1/admin/urls/{code} [delete]
func (s *Server) deleteURLHandler(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "admin.DeleteURLHandler")
	defer span.End()

	params := new(DeleteURLParams)
	if err := c.Bind(params); err != nil {
		span.SetStatus(codes.Error, "failed to bind request")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(params); err != nil {
		return s.failedValidationError(c, err)
	}
	span.SetAttributes(attribute.String("code", params.Code))

	rowsAffected, err := s.rep.DeleteURL(ctx, params.Code)
	if err != nil {
		span.SetStatus(codes.Error, "failed to delete url")
		span.RecordError(err)

		s.logger.Error().Err(err).Str("code", params.Code).Msg("failed to delete short url")
		return echo.ErrInternalServerError
	}
	if rowsAffected == 0 {
		span.AddEvent("short url not found", trace.WithAttributes(attribute.String("code", params.Code)))
		s.logger.Warn().Str("code", params.Code).Msg("short url not found")
		return echo.ErrNotFound
	}

	if removedKeys, err := s.cache.DeleteLongURL(ctx, params.Code); err != nil {
		span.AddEvent("failed to delete long url from cache", trace.WithAttributes(attribute.String("code", params.Code), attribute.Int64("removedKeys", removedKeys)))
		s.logger.Warn().Err(err).Str("code", params.Code).Int64("removedKeys", removedKeys).Msg("failed to delete long url from cache")
	}

	return c.NoContent(http.StatusNoContent)
}

type DeleteUserURLsParams struct {
	UserID string `param:"userId" validate:"required,min=1,max=50"`
}
type DeleteUserURLsResponse struct {
	Deleted int `json:"deleted"`
}

// deleteUserURLs godoc
//
//	@Summary		Delete URLs created by a user
//	@Description	Delete all URLs created by a user. Also removes them from cache.
//	@Tags			Admin
//	@Produce		json
//	@Param			userId	path		string					true	"ID of the user"	minlength(1)	maxlength(50)
//	@Success		200		{object}	DeleteUserURLsResponse	"Number of URLs deleted"
//	@Failure		400		{object}	HTTPValidationError		"Validation failed"
//	@Failure		401		{object}	HTTPError				"Unauthorized"
//	@Failure		403		{object}	HTTPError				"Forbidden"
//	@Failure		500		{object}	HTTPError				"Internal server error"
//	@Security		BearerAuth
//	@Router			/v1/admin/urls/user/{userId} [delete]
func (s *Server) deleteUserURLsHandler(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "admin.DeleteUserURLsHandler")
	defer span.End()

	params := new(DeleteUserURLsParams)
	if err := c.Bind(params); err != nil {
		span.SetStatus(codes.Error, "failed to bind request")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(params); err != nil {
		return s.failedValidationError(c, err)
	}
	span.SetAttributes(attribute.String("userId", params.UserID))

	deletedIDs, err := s.rep.DeleteAllUserURLs(ctx, params.UserID)
	if err != nil {
		span.SetStatus(codes.Error, "failed to delete user urls")
		span.RecordError(err)

		s.logger.Error().Err(err).Str("userId", params.UserID).Msg("failed to delete user urls")
		return echo.ErrInternalServerError
	}

	if removedKeys, err := s.cache.DeleteLongURLs(ctx, deletedIDs); err != nil {
		span.AddEvent("failed to delete user urls from cache", trace.WithAttributes(attribute.String("userId", params.UserID), attribute.Int64("removedKeys", removedKeys), attribute.StringSlice("deletedIDs", deletedIDs)))
		s.logger.Warn().Err(err).Str("userId", params.UserID).Int64("removedKeys", removedKeys).Strs("deletedIDs", deletedIDs).Msg("failed to delete user urls from cache")
	}

	return c.JSON(http.StatusOK, &DeleteUserURLsResponse{
		Deleted: len(deletedIDs),
	})
}

type BlockUserDTO struct {
	Reason *string `json:"reason" validate:"omitzero,min=1,max=255"`
}
type BlockUserParams struct {
	DeleteUserURLsParams
	BlockUserDTO
}

// blockUser godoc
//
//	@Summary		Block a user
//	@Description	Block a user in the system, preventing them from accessing their account.
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			userId	path		string					true	"ID of the user"	minlength(1)	maxlength(50)
//	@Param			request	body		BlockUserDTO			true	"Block user request body"
//	@Success		201		{object}	repository.UserBlock	"User Block entity"
//	@Failure		400		{object}	HTTPValidationError		"Validation failed"
//	@Failure		401		{object}	HTTPError				"Unauthorized"
//	@Failure		403		{object}	HTTPError				"Forbidden"
//	@Failure		404		{object}	HTTPError				"Not Found"
//	@Failure		500		{object}	HTTPError				"Internal server error"
//	@Security		BearerAuth
//	@Router			/v1/admin/users/block/{userId} [post]
func (s *Server) blockUserHandler(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "admin.BlockUserHandler")
	defer span.End()

	params := new(BlockUserParams)
	if err := c.Bind(params); err != nil {
		span.SetStatus(codes.Error, "failed to bind request")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(params); err != nil {
		return s.failedValidationError(c, err)
	}

	span.SetAttributes(attribute.String("userId", params.UserID))
	if params.Reason != nil {
		span.SetAttributes(attribute.String("reason", *params.Reason))
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		span.SetStatus(codes.Error, "failed to start transaction")
		span.RecordError(err)
		s.logger.Error().Err(err).Msg("failed to start transaction")

		return echo.ErrInternalServerError
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	userId := auth.GetUserID(c)
	qtx := s.rep.WithTx(tx)

	// Block the user in Auth0 first
	updatedUser, err := s.authManagement.BlockUser(ctx, params.UserID)
	if err != nil {
		span.SetStatus(codes.Error, "failed to block the user in auth0")

		var apiErr *core.APIError
		if errors.As(err, &apiErr) {
			span.RecordError(apiErr.Unwrap())

			s.logger.Error().Err(apiErr.Unwrap()).Str("userId", params.UserID).Msg("failed to block the user in auth0")
			return echo.NewHTTPError(apiErr.StatusCode)
		}

		span.RecordError(err)
		s.logger.Error().Err(err).Str("userId", params.UserID).Msg("failed to block the user in auth0")

		return echo.ErrInternalServerError
	}

	// Block the user in the DB
	userBlock, err := qtx.BlockUser(ctx, repository.BlockUserParams{UserID: params.UserID, UserEmail: updatedUser.Email, BlockedBy: *userId, Reason: params.Reason})
	if err != nil {
		span.SetStatus(codes.Error, "failed to block the user in db")
		span.RecordError(err)
		s.logger.Error().Err(err).Str("userId", params.UserID).Msg("failed to block the user in db")

		_ = s.authManagement.UnblockUser(ctx, params.UserID)

		return echo.ErrInternalServerError
	}

	if err := tx.Commit(ctx); err != nil {
		span.SetStatus(codes.Error, "failed to commit transaction")
		span.RecordError(err)
		s.logger.Error().Err(err).Msg("failed to commit transaction")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusCreated, userBlock)
}

// unblockUser godoc
//
//	@Summary		Unblock a user
//	@Description	Unblock a user in the system, allowing them to access their account.
//	@Tags			Admin
//	@Produce		json
//	@Param			userId	path		string					true	"ID of the user"	minlength(1)	maxlength(50)
//	@Success		200		{object}	repository.UserBlock	"Updated User Block entity"
//	@Failure		400		{object}	HTTPValidationError		"Validation failed"
//	@Failure		401		{object}	HTTPError				"Unauthorized"
//	@Failure		403		{object}	HTTPError				"Forbidden"
//	@Failure		404		{object}	HTTPError				"Not Found"
//	@Failure		500		{object}	HTTPError				"Internal server error"
//	@Security		BearerAuth
//	@Router			/v1/admin/users/unblock/{userId} [post]
func (s *Server) unblockUserHandler(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "admin.UnblockUserHandler")
	defer span.End()

	params := new(BlockUserParams)
	if err := c.Bind(params); err != nil {
		span.SetStatus(codes.Error, "failed to bind request")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(params); err != nil {
		return s.failedValidationError(c, err)
	}
	span.SetAttributes(attribute.String("userId", params.UserID))

	tx, err := s.db.Begin(ctx)
	if err != nil {
		span.SetStatus(codes.Error, "failed to start transaction")
		span.RecordError(err)
		s.logger.Error().Err(err).Msg("failed to start transaction")

		return echo.ErrInternalServerError
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	userId := auth.GetUserID(c)
	qtx := s.rep.WithTx(tx)

	// For unblock, we don't need Auth0 response,
	// so first, update the DB record, then unblock the user in Auth0.
	// And if DB update fails, do not make any Auth0 calls
	userBlock, err := qtx.UnblockUser(ctx, repository.UnblockUserParams{UserID: params.UserID, UnblockedBy: *userId})
	if err != nil {
		span.SetStatus(codes.Error, "failed to unblock the user in db")
		span.RecordError(err)

		if s.rep.IsNotFoundError(err) {
			s.logger.Error().Err(err).Str("userId", params.UserID).Msg("user block not found")
			return echo.ErrNotFound
		}

		s.logger.Error().Err(err).Str("userId", params.UserID).Msg("failed to unblock the user in db")
		return echo.ErrInternalServerError
	}

	err = s.authManagement.UnblockUser(ctx, params.UserID)
	if err != nil {
		span.SetStatus(codes.Error, "failed to unblock the user in auth")

		var apiErr *core.APIError
		if errors.As(err, &apiErr) {
			span.RecordError(apiErr.Unwrap())

			s.logger.Error().Err(apiErr.Unwrap()).Str("userId", params.UserID).Msg("failed to unblock the user in auth")
			return echo.NewHTTPError(apiErr.StatusCode)
		}

		span.RecordError(err)
		s.logger.Error().Err(err).Str("userId", params.UserID).Msg("failed to unblock the user in auth")

		return echo.ErrInternalServerError
	}

	if err := tx.Commit(ctx); err != nil {
		span.SetStatus(codes.Error, "failed to commit transaction")
		span.RecordError(err)
		s.logger.Error().Err(err).Msg("failed to commit transaction")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, userBlock)
}

type UserBlocksFilters struct {
	PaginationFilters
}
type PaginatedUserBlocks struct {
	Items      []repository.UserBlock `json:"items"`
	Pagination Pagination             `json:"pagination"`
}

// getUserBlocks godoc
//
//	@Summary		Get all User Blocks
//	@Description	Retrieves a paginated list of all User Blocks created by admins
//	@Tags			Admin
//	@Produce		json
//	@Param			page		query		int					true	"Page number"	minimum(1)	maximum(10000)	default(1)
//	@Param			pageSize	query		int					true	"Page size"		minimum(1)	maximum(100)	default(20)
//	@Success		200			{object}	PaginatedUserBlocks	"Paginated list of User Blocks"
//	@Failure		400			{object}	HTTPValidationError	"Validation failed"
//	@Failure		401			{object}	HTTPError			"Unauthorized"
//	@Failure		403			{object}	HTTPError			"Forbidden"
//	@Failure		500			{object}	HTTPError			"Internal server error"
//	@Security		BearerAuth
//	@Router			/v1/admin/users/blocks [get]
func (s *Server) getUserBlocks(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "admin.GetUserBlocks")
	defer span.End()

	params := new(UserBlocksFilters)
	if err := c.Bind(params); err != nil {
		span.SetStatus(codes.Error, "failed to bind request")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(params); err != nil {
		return s.failedValidationError(c, err)
	}

	span.SetAttributes(attribute.Int("page", int(params.Page)), attribute.Int("pageSize", int(params.PageSize)))

	userBlocks, err := s.rep.GetUserBlocks(ctx, repository.GetUserBlocksParams{Limit: params.limit(), Offset: params.offset()})
	if err != nil {
		span.SetStatus(codes.Error, "failed to get user blocks")
		span.RecordError(err)

		return echo.ErrInternalServerError
	}

	var totalCount int
	if len(userBlocks) > 0 {
		totalCount = int(userBlocks[0].TotalCount)
	}

	items := make([]repository.UserBlock, len(userBlocks))
	for i, userBlock := range userBlocks {
		items[i] = repository.UserBlock{
			ID:          userBlock.ID,
			UserID:      userBlock.UserID,
			UserEmail:   userBlock.UserEmail,
			BlockedBy:   userBlock.BlockedBy,
			BlockedAt:   userBlock.BlockedAt,
			UnblockedBy: userBlock.UnblockedBy,
			UnblockedAt: userBlock.UnblockedAt,
			Reason:      userBlock.Reason,
		}
	}

	response := &PaginatedUserBlocks{
		Items:      items,
		Pagination: calculatePagination(totalCount, int(params.Page), int(params.PageSize)),
	}

	return c.JSON(http.StatusOK, response)
}
