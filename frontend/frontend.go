package frontend

import (
	"log/slog"
	"net/http"
	"time"

	"bakery/frontend/internal/backend"
	"bakery/frontend/internal/web"
)

func NewHandler(api http.Handler, apiTimeout time.Duration, logger *slog.Logger) (http.Handler, error) {
	client := backend.NewInProcess(api, apiTimeout)
	return web.New(client, client, logger)
}
