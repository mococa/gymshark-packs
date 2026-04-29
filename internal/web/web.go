package web

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/mococa/gymshark-packs/internal/store"
	"github.com/mococa/gymshark-packs/internal/web/templates"
)

//go:embed all:static
var staticFS embed.FS

type Handler struct {
	store  store.PackSizeStore
	logger *slog.Logger
}

func New(store store.PackSizeStore, logger *slog.Logger) *Handler {
	return &Handler{store: store, logger: logger}
}

func (h *Handler) ServeHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		h.ServeNotFound(w, r)
		return
	}

	packSizes, err := h.store.GetAll()
	if err != nil {
		h.logger.Error("failed to load pack sizes for home page", "error", err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.Home(packSizes).Render(r.Context(), w)
}

func (h *Handler) ServeNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	templates.NotFound().Render(r.Context(), w)
}

func StaticFS() http.FileSystem {
	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return http.FS(subFS)
}
