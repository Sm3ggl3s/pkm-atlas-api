package seed

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"sync"
	"testing"

	"github.com/Sm3ggl3s/pkm-atlas-api/internal/pokeapi"
	"github.com/Sm3ggl3s/pkm-atlas-api/internal/pokemon"
)

// --- canned PokéAPI fixtures (two-stage Bulbasaur -> Ivysaur line) ---

const (
	bulbasaurSpeciesJSON = `{
		"id": 1, "name": "bulbasaur", "is_legendary": false, "is_mythical": false, "is_baby": false,
		"names": [{"language": {"name": "en"}, "name": "Bulbasaur"}],
		"generation": {"name": "generation-i", "url": "https://x/api/v2/generation/1/"},
		"flavor_text_entries": [
			{"flavor_text": "A strange\nseed.", "language": {"name": "fr"}, "version": {"name": "x"}},
			{"flavor_text": "A strange\nseed\fwas planted.", "language": {"name": "en"}, "version": {"name": "red"}}
		],
		"evolution_chain": {"url": "https://x/api/v2/evolution-chain/1/"},
		"varieties": [{"is_default": true, "pokemon": {"name": "bulbasaur", "url": "https://x/api/v2/pokemon/1/"}}],
		"pokedex_numbers": [{"entry_number": 1, "pokedex": {"name": "national"}}]
	}`
	ivysaurSpeciesJSON = `{
		"id": 2, "name": "ivysaur",
		"names": [{"language": {"name": "en"}, "name": "Ivysaur"}],
		"generation": {"name": "generation-i", "url": "https://x/api/v2/generation/1/"},
		"flavor_text_entries": [],
		"evolution_chain": {"url": "https://x/api/v2/evolution-chain/1/"},
		"varieties": [{"is_default": true, "pokemon": {"name": "ivysaur", "url": "https://x/api/v2/pokemon/2/"}}],
		"pokedex_numbers": [{"entry_number": 2, "pokedex": {"name": "national"}}]
	}`
	bulbasaurPokemonJSON = `{"id": 1, "name": "bulbasaur", "height": 7, "weight": 69, "is_default": true,
		"types": [{"slot": 2, "type": {"name": "poison"}}, {"slot": 1, "type": {"name": "grass"}}]}`
	ivysaurPokemonJSON = `{"id": 2, "name": "ivysaur", "height": 10, "weight": 130, "is_default": true,
		"types": [{"slot": 1, "type": {"name": "grass"}}, {"slot": 2, "type": {"name": "poison"}}]}`
	chainJSON = `{"id": 1, "chain": {
		"species": {"name": "bulbasaur"},
		"evolves_to": [{"species": {"name": "ivysaur"}, "evolves_to": []}]
	}}`
)

func newFakeClient(t *testing.T) *fakeClient {
	t.Helper()
	return &fakeClient{
		refs: []pokeapi.NamedRef{
			{Name: "bulbasaur", URL: "https://x/api/v2/pokemon-species/1/"},
			{Name: "ivysaur", URL: "https://x/api/v2/pokemon-species/2/"},
		},
		species: map[string]*pokeapi.Species{
			"https://x/api/v2/pokemon-species/1/": unmarshal[pokeapi.Species](t, bulbasaurSpeciesJSON),
			"https://x/api/v2/pokemon-species/2/": unmarshal[pokeapi.Species](t, ivysaurSpeciesJSON),
		},
		pokemon: map[string]*pokeapi.Pokemon{
			"https://x/api/v2/pokemon/1/": unmarshal[pokeapi.Pokemon](t, bulbasaurPokemonJSON),
			"https://x/api/v2/pokemon/2/": unmarshal[pokeapi.Pokemon](t, ivysaurPokemonJSON),
		},
		chains: map[string]*pokeapi.EvolutionChain{
			"https://x/api/v2/evolution-chain/1/": unmarshal[pokeapi.EvolutionChain](t, chainJSON),
		},
	}
}

func unmarshal[T any](t *testing.T, s string) *T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return &v
}

type fakeClient struct {
	refs    []pokeapi.NamedRef
	species map[string]*pokeapi.Species
	pokemon map[string]*pokeapi.Pokemon
	chains  map[string]*pokeapi.EvolutionChain
}

func (f *fakeClient) ListSpeciesRefs(context.Context) ([]pokeapi.NamedRef, error) { return f.refs, nil }
func (f *fakeClient) GetSpecies(_ context.Context, url string) (*pokeapi.Species, error) {
	return f.species[url], nil
}
func (f *fakeClient) GetPokemon(_ context.Context, url string) (*pokeapi.Pokemon, error) {
	return f.pokemon[url], nil
}
func (f *fakeClient) GetEvolutionChain(_ context.Context, url string) (*pokeapi.EvolutionChain, error) {
	return f.chains[url], nil
}

// fakeStore emulates upsert semantics: species keyed by id, evolutions replaced.
type fakeStore struct {
	mu         sync.Mutex
	species    map[int]pokemon.SeedSpecies
	evolutions []pokemon.SeedEvolution
}

func newFakeStore() *fakeStore { return &fakeStore{species: map[int]pokemon.SeedSpecies{}} }

func (s *fakeStore) TypeIDsBySlug(context.Context) (map[string]int, error) {
	return map[string]int{"grass": 5, "poison": 8}, nil
}
func (s *fakeStore) UpsertSpecies(_ context.Context, sp pokemon.SeedSpecies) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.species[sp.ID] = sp
	return nil
}
func (s *fakeStore) ReplaceEvolutions(_ context.Context, edges []pokemon.SeedEvolution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evolutions = edges
	return nil
}

func quietLogger() *log.Logger { return log.New(io.Discard, "", 0) }

func TestSeederMapsAndWrites(t *testing.T) {
	store := newFakeStore()
	seeder := NewSeeder(newFakeClient(t), store, quietLogger())

	summary, err := seeder.Run(context.Background(), Options{Concurrency: 1})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if summary.Species != 2 || summary.Forms != 2 || summary.Evolutions != 1 {
		t.Errorf("summary = %+v, want {2 2 1}", summary)
	}

	bulba, ok := store.species[1]
	if !ok {
		t.Fatal("bulbasaur (id 1) was not upserted")
	}
	if bulba.NationalDexNumber != 1 || bulba.Name != "Bulbasaur" || bulba.Slug != "bulbasaur" {
		t.Errorf("bulbasaur identity wrong: %+v", bulba)
	}
	if bulba.GenerationID != 1 {
		t.Errorf("generation id = %d, want 1 (parsed from URL)", bulba.GenerationID)
	}
	if bulba.Description == nil || *bulba.Description != "A strange seed was planted." {
		t.Errorf("description = %v, want cleaned english text", bulba.Description)
	}
	if len(bulba.Forms) != 1 {
		t.Fatalf("forms = %d, want 1", len(bulba.Forms))
	}
	f := bulba.Forms[0]
	if f.Height == nil || *f.Height != 7 || f.Weight == nil || *f.Weight != 69 || !f.IsDefault {
		t.Errorf("form fields wrong: %+v", f)
	}
	if len(f.TypeIDs) != 2 || f.TypeIDs[0] != 5 || f.TypeIDs[1] != 8 {
		t.Errorf("type ids = %v, want [5 8] (grass, poison in slot order)", f.TypeIDs)
	}

	if len(store.evolutions) != 1 ||
		store.evolutions[0] != (pokemon.SeedEvolution{FromSpeciesID: 1, ToSpeciesID: 2}) {
		t.Errorf("evolutions = %+v, want one edge 1->2", store.evolutions)
	}
}

func TestSeederIsIdempotent(t *testing.T) {
	store := newFakeStore()
	seeder := NewSeeder(newFakeClient(t), store, quietLogger())

	for i := 0; i < 2; i++ {
		if _, err := seeder.Run(context.Background(), Options{Concurrency: 4}); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}

	if len(store.species) != 2 {
		t.Errorf("species count = %d after two runs, want 2 (upsert, not duplicate)", len(store.species))
	}
	if len(store.evolutions) != 1 {
		t.Errorf("evolutions = %d after two runs, want 1", len(store.evolutions))
	}
}
