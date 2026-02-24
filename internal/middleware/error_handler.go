package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func MapDBError(err error) (int, ErrorResponse) {
	if errors.Is(err, pgx.ErrNoRows) {
		return http.StatusNotFound, ErrorResponse{Error: "resource not found"}
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return http.StatusConflict, ErrorResponse{
				Error:   "resource already exists",
				Details: pgErr.Detail,
			}
		case "23503": // foreign_key_violation
			return http.StatusBadRequest, ErrorResponse{
				Error:   "referenced resource does not exist",
				Details: pgErr.Detail,
			}
		case "23514": // check_violation
			return http.StatusBadRequest, ErrorResponse{
				Error:   "constraint violation",
				Details: pgErr.Detail,
			}
		case "23P01": // exclusion_violation
			return http.StatusConflict, ErrorResponse{
				Error:   "overlapping resource",
				Details: pgErr.Detail,
			}
		}
	}

	log.Error().Err(err).Msg("unhandled database error")
	return http.StatusInternalServerError, ErrorResponse{Error: "internal server error"}
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			status, resp := MapDBError(err)
			c.JSON(status, resp)
		}
	}
}
