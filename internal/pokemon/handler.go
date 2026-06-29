package pokemon

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// repository is the subset of the data layer the handlers depend on. Defining
// the interface at the point of use keeps the handlers thin and lets tests
// supply a fake without a database.
type repository interface {
	ListPokemon(ctx context.Context) ([]ListItem, error)
	GetPokemonBySlug(ctx context.Context, slug string) (*Detail, error)
	ListTypes(ctx context.Context) ([]TypeResponse, error)
	ListPokemonByType(ctx context.Context, typeSlug string) ([]ListItem, error)
}

// Handler holds the dependencies for the Pokédex HTTP endpoints.
type Handler struct {
	repo repository
}

// NewHandler returns a Handler backed by the given repository.
func NewHandler(repo repository) *Handler {
	return &Handler{repo: repo}
}

// ListPokemon handles GET /api/v1/pokemon.
func (h *Handler) ListPokemon(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListPokemon(r.Context())
	if err != nil {
		log.Printf("ListPokemon: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list pokemon")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// GetPokemon handles GET /api/v1/pokemon/{slug}.
func (h *Handler) GetPokemon(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	p, err := h.repo.GetPokemonBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "pokemon not found")
			return
		}
		log.Printf("GetPokemon %q: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "failed to get pokemon")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// ListTypes handles GET /api/v1/types.
func (h *Handler) ListTypes(w http.ResponseWriter, r *http.Request) {
	types, err := h.repo.ListTypes(r.Context())
	if err != nil {
		log.Printf("ListTypes: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list types")
		return
	}
	writeJSON(w, http.StatusOK, types)
}

// ListPokemonByType handles GET /api/v1/types/{slug}/pokemon.
func (h *Handler) ListPokemonByType(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	items, err := h.repo.ListPokemonByType(r.Context(), slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "type not found")
			return
		}
		log.Printf("ListPokemonByType %q: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "failed to list pokemon by type")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// writeJSON writes status and body as JSON.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write json response: %v", err)
	}
}

// writeError writes a simple {"error": msg} JSON body with the given status.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
