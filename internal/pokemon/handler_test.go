package pokemon

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeRepo implements the repository interface with canned returns.
type fakeRepo struct {
	list     []ListItem
	detail   *Detail
	types    []TypeResponse
	byType   []ListItem
	err      error // returned by every method when non-nil
	notFound bool  // detail/byType return ErrNotFound when true
}

func (f *fakeRepo) ListPokemon(context.Context) ([]ListItem, error) {
	return f.list, f.err
}
func (f *fakeRepo) GetPokemonBySlug(_ context.Context, _ string) (*Detail, error) {
	if f.notFound {
		return nil, ErrNotFound
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.detail, nil
}
func (f *fakeRepo) ListTypes(context.Context) ([]TypeResponse, error) {
	return f.types, f.err
}
func (f *fakeRepo) ListPokemonByType(_ context.Context, _ string) ([]ListItem, error) {
	if f.notFound {
		return nil, ErrNotFound
	}
	return f.byType, f.err
}

func doRequest(t *testing.T, h http.HandlerFunc, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestListPokemonOK(t *testing.T) {
	h := NewHandler(&fakeRepo{list: []ListItem{
		{ID: 1, NationalDexNumber: 1, Name: "Bulbasaur", Slug: "bulbasaur", Types: []string{"grass", "poison"}},
	}})

	rec := doRequest(t, h.ListPokemon, "/api/v1/pokemon")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var got []ListItem
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].Slug != "bulbasaur" || len(got[0].Types) != 2 {
		t.Errorf("unexpected body: %+v", got)
	}
}

func TestListPokemonInternalError(t *testing.T) {
	h := NewHandler(&fakeRepo{err: errors.New("boom")})
	rec := doRequest(t, h.ListPokemon, "/api/v1/pokemon")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestGetPokemonOK(t *testing.T) {
	h := NewHandler(&fakeRepo{detail: &Detail{
		ID: 1, NationalDexNumber: 1, Name: "Bulbasaur", Slug: "bulbasaur",
		Types: []string{"grass", "poison"}, EvolvesTo: []string{"ivysaur"},
	}})

	// PathValue is empty without a router, but the handler still serves detail.
	rec := doRequest(t, h.GetPokemon, "/api/v1/pokemon/bulbasaur")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got Detail
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Slug != "bulbasaur" || len(got.EvolvesTo) != 1 {
		t.Errorf("unexpected body: %+v", got)
	}
}

func TestGetPokemonNotFound(t *testing.T) {
	h := NewHandler(&fakeRepo{notFound: true})
	rec := doRequest(t, h.GetPokemon, "/api/v1/pokemon/missingno")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestListTypesOK(t *testing.T) {
	h := NewHandler(&fakeRepo{types: []TypeResponse{{ID: 1, Name: "Normal", Slug: "normal"}}})
	rec := doRequest(t, h.ListTypes, "/api/v1/types")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestListPokemonByTypeOK(t *testing.T) {
	h := NewHandler(&fakeRepo{byType: []ListItem{
		{ID: 4, NationalDexNumber: 4, Name: "Charmander", Slug: "charmander", Types: []string{"fire"}},
	}})
	rec := doRequest(t, h.ListPokemonByType, "/api/v1/types/fire/pokemon")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestListPokemonByTypeNotFound(t *testing.T) {
	h := NewHandler(&fakeRepo{notFound: true})
	rec := doRequest(t, h.ListPokemonByType, "/api/v1/types/nope/pokemon")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
