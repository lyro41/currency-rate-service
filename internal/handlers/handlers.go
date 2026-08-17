package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/go-chi/render"

	"github.com/lyro41/currency-rate-service/internal/api"
)

var pairRe = regexp.MustCompile(`^([a-zA-Z]+)/([a-zA-Z]+)$`)

func handlePair(w http.ResponseWriter, r *http.Request, resp *api.ErrorResponse, logger *slog.Logger) bool {
	if !pairRe.MatchString(resp.Pair) {
		resp.Error = "'pair' parameter must be in 'ABC/XYZ' format"
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, resp)
		logger.Warn("invalid pair")
		return false
	}
	currencies := pairRe.FindStringSubmatch(resp.Pair)
	for _, currency := range currencies[1:] {
		if !validCurrencies[currency] {
			resp.Error = fmt.Sprintf("currency %s is not supported", currency)
			render.Status(r, http.StatusUnprocessableEntity)
			render.JSON(w, r, resp)
			logger.Warn("unsupported currency in pair")
			return false
		}
	}
	return true
}

func writeError(w http.ResponseWriter, r *http.Request, status int, resp *api.ErrorResponse) {
	render.Status(r, status)
	render.JSON(w, r, resp)
}
