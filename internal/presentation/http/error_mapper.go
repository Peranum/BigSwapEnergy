package http

import (
	"encoding/json"

	apperrors "bigswapenergy/internal/shared/errors"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type ErrorMapping struct {
	HTTPStatus int
	Code       string
	Message    string
	ShouldLog  bool
}

var errorMappings = map[error]ErrorMapping{
	apperrors.ErrValidation: {
		HTTPStatus: fasthttp.StatusBadRequest,
		Code:       "VALIDATION_ERROR",
		Message:    "Request validation failed",
		ShouldLog:  false,
	},
	apperrors.ErrInvalidInput: {
		HTTPStatus: fasthttp.StatusBadRequest,
		Code:       "INVALID_INPUT",
		Message:    "Invalid input parameters",
		ShouldLog:  false,
	},

	apperrors.ErrBusinessRule: {
		HTTPStatus: fasthttp.StatusBadRequest,
		Code:       "BUSINESS_RULE_VIOLATION",
		Message:    "Business rule violation",
		ShouldLog:  false,
	},
	apperrors.ErrNotFound: {
		HTTPStatus: fasthttp.StatusNotFound,
		Code:       "NOT_FOUND",
		Message:    "Requested resource not found",
		ShouldLog:  false,
	},

	apperrors.ErrExternalService: {
		HTTPStatus: fasthttp.StatusBadGateway,
		Code:       "EXTERNAL_SERVICE_ERROR",
		Message:    "External service unavailable",
		ShouldLog:  true,
	},
	apperrors.ErrTimeout: {
		HTTPStatus: fasthttp.StatusGatewayTimeout,
		Code:       "TIMEOUT_ERROR",
		Message:    "Request timeout",
		ShouldLog:  true,
	},

	apperrors.ErrInternal: {
		HTTPStatus: fasthttp.StatusInternalServerError,
		Code:       "INTERNAL_ERROR",
		Message:    "Internal server error",
		ShouldLog:  true,
	},
}

func (h *EstimateHandler) handleError(ctx *fasthttp.RequestCtx, err error) {
	mapping, found := errorMappings[err]

	if !found {
		mapping = ErrorMapping{
			HTTPStatus: fasthttp.StatusInternalServerError,
			Code:       "UNKNOWN_ERROR",
			Message:    "An unexpected error occurred",
			ShouldLog:  true,
		}
	}

	if mapping.ShouldLog {
		h.logger.Error("Request error",
			zap.Error(err),
			zap.String("path", string(ctx.Path())),
			zap.String("method", string(ctx.Method())),
			zap.String("code", mapping.Code))
	}

	errorResp := ErrorResponse{
		Code:    mapping.Code,
		Message: mapping.Message,
		Details: getErrorDetails(err, mapping.HTTPStatus >= 500),
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(mapping.HTTPStatus)
	json.NewEncoder(ctx).Encode(map[string]ErrorResponse{"error": errorResp})
}

func getErrorDetails(err error, isServerError bool) string {
	if isServerError {
		return ""
	}
	return err.Error()
}
