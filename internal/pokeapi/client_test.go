package pokeapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func TestGetPokemonParsesFields(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": 6, "name": "charizard", "height": 17, "weight": 905, "is_default": true,
			"types": [
				{"slot": 2, "type": {"name": "flying", "url": "/type/3/"}},
				{"slot": 1, "type": {"name": "fire", "url": "/type/10/"}}
			]
		}`))
	})

	client := NewClient(srv.URL, srv.Client())
	p, err := client.GetPokemon(context.Background(), srv.URL+"/pokemon/6")
	if err != nil {
		t.Fatalf("GetPokemon: %v", err)
	}

	if p.ID != 6 || p.Name != "charizard" || p.Height != 17 || p.Weight != 905 || !p.IsDefault {
		t.Errorf("unexpected pokemon: %+v", p)
	}
	if len(p.Types) != 2 {
		t.Fatalf("types = %d, want 2", len(p.Types))
	}
}

func TestListSpeciesRefs(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"count": 2, "results": [
			{"name": "bulbasaur", "url": "/pokemon-species/1/"},
			{"name": "ivysaur", "url": "/pokemon-species/2/"}
		]}`))
	})

	client := NewClient(srv.URL, srv.Client())
	refs, err := client.ListSpeciesRefs(context.Background())
	if err != nil {
		t.Fatalf("ListSpeciesRefs: %v", err)
	}
	if len(refs) != 2 || refs[0].Name != "bulbasaur" {
		t.Errorf("unexpected refs: %+v", refs)
	}
}

func TestGetPokemonNotFoundFailsFast(t *testing.T) {
	var calls int
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusNotFound)
	})

	client := NewClient(srv.URL, srv.Client())
	if _, err := client.GetPokemon(context.Background(), srv.URL+"/pokemon/99999"); err == nil {
		t.Fatal("expected an error for 404, got nil")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry on 404), got %d", calls)
	}
}
