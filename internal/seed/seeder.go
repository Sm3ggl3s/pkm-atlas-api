// Package seed orchestrates importing the Pokédex from PokéAPI into Postgres.
// It fetches via a pokeapi.Client, maps the responses onto our seed models, and
// upserts them through a store so the database becomes a 1-1, re-runnable mirror
// of PokéAPI.
package seed

import (
	"context"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Sm3ggl3s/pkm-atlas-api/internal/pokeapi"
	"github.com/Sm3ggl3s/pkm-atlas-api/internal/pokemon"
)

// Client is the subset of pokeapi.Client the seeder needs (an interface so tests
// can supply a fake).
type Client interface {
	ListSpeciesRefs(ctx context.Context) ([]pokeapi.NamedRef, error)
	GetSpecies(ctx context.Context, url string) (*pokeapi.Species, error)
	GetPokemon(ctx context.Context, url string) (*pokeapi.Pokemon, error)
	GetEvolutionChain(ctx context.Context, url string) (*pokeapi.EvolutionChain, error)
}

// Store is the subset of the write layer the seeder needs.
type Store interface {
	TypeIDsBySlug(ctx context.Context) (map[string]int, error)
	UpsertSpecies(ctx context.Context, sp pokemon.SeedSpecies) error
	ReplaceEvolutions(ctx context.Context, edges []pokemon.SeedEvolution) error
}

// Options tunes a seeding run.
type Options struct {
	Limit       int // max species to import; 0 means all
	Concurrency int // parallel PokéAPI fetches; <1 means 1
}

// Summary reports what a run wrote.
type Summary struct {
	Species    int
	Forms      int
	Evolutions int
}

// Seeder imports the dex from a Client into a Store.
type Seeder struct {
	client Client
	store  Store
	log    *log.Logger
}

// NewSeeder builds a Seeder. A nil logger falls back to the standard logger.
func NewSeeder(client Client, store Store, logger *log.Logger) *Seeder {
	if logger == nil {
		logger = log.Default()
	}
	return &Seeder{client: client, store: store, log: logger}
}

// Run imports species + forms (phase 1) then evolution edges (phase 2).
func (s *Seeder) Run(ctx context.Context, opts Options) (Summary, error) {
	typeMap, err := s.store.TypeIDsBySlug(ctx)
	if err != nil {
		return Summary{}, err
	}

	refs, err := s.client.ListSpeciesRefs(ctx)
	if err != nil {
		return Summary{}, err
	}
	if opts.Limit > 0 && opts.Limit < len(refs) {
		refs = refs[:opts.Limit]
	}

	// Shared, mutex-guarded collectors written by the concurrent workers.
	var (
		mu        sync.Mutex
		formCount int
		nameToID  = make(map[string]int, len(refs))
		chainURLs = make(map[string]struct{})
	)

	err = parallel(ctx, refs, opts.Concurrency, func(ctx context.Context, ref pokeapi.NamedRef) error {
		species, err := s.client.GetSpecies(ctx, ref.URL)
		if err != nil {
			return err
		}

		sp, forms, err := s.buildSpecies(ctx, species, typeMap)
		if err != nil {
			return err
		}
		if err := s.store.UpsertSpecies(ctx, sp); err != nil {
			return err
		}

		mu.Lock()
		formCount += forms
		nameToID[species.Name] = sp.ID
		if url := species.EvolutionChain.URL; url != "" {
			chainURLs[url] = struct{}{}
		}
		mu.Unlock()
		return nil
	})
	if err != nil {
		return Summary{}, err
	}

	edges, err := s.collectEvolutions(ctx, chainURLs, nameToID, opts.Concurrency)
	if err != nil {
		return Summary{}, err
	}
	if err := s.store.ReplaceEvolutions(ctx, edges); err != nil {
		return Summary{}, err
	}

	return Summary{Species: len(refs), Forms: formCount, Evolutions: len(edges)}, nil
}

// buildSpecies maps a PokéAPI species (and its varieties) onto a SeedSpecies,
// returning the number of forms it contains.
func (s *Seeder) buildSpecies(
	ctx context.Context, species *pokeapi.Species, typeMap map[string]int,
) (pokemon.SeedSpecies, int, error) {
	sp := pokemon.SeedSpecies{
		ID:                species.ID,
		NationalDexNumber: nationalDexNumber(species),
		Name:              displayName(species),
		Slug:              species.Name,
		GenerationID:      parseTrailingID(species.Generation.URL),
		IsLegendary:       species.IsLegendary,
		IsMythical:        species.IsMythical,
		IsBaby:            species.IsBaby,
		Description:       englishDescription(species),
	}

	for _, v := range species.Varieties {
		p, err := s.client.GetPokemon(ctx, v.Pokemon.URL)
		if err != nil {
			return pokemon.SeedSpecies{}, 0, err
		}
		sp.Forms = append(sp.Forms, s.buildForm(p, typeMap))
	}
	return sp, len(sp.Forms), nil
}

// buildForm maps a PokéAPI pokemon (variety) onto a SeedForm, resolving type
// slugs to ids in slot order.
func (s *Seeder) buildForm(p *pokeapi.Pokemon, typeMap map[string]int) pokemon.SeedForm {
	type slotType struct{ slot, id int }
	resolved := make([]slotType, 0, len(p.Types))
	for _, t := range p.Types {
		id, ok := typeMap[t.Type.Name]
		if !ok {
			s.log.Printf("seed: unknown type %q on %q, skipping", t.Type.Name, p.Name)
			continue
		}
		resolved = append(resolved, slotType{slot: t.Slot, id: id})
	}
	sort.Slice(resolved, func(i, j int) bool { return resolved[i].slot < resolved[j].slot })

	typeIDs := make([]int, len(resolved))
	for i, rt := range resolved {
		typeIDs[i] = rt.id
	}

	return pokemon.SeedForm{
		Slug:      p.Name,
		Name:      p.Name,
		IsDefault: p.IsDefault,
		Height:    positiveOrNil(p.Height),
		Weight:    positiveOrNil(p.Weight),
		TypeIDs:   typeIDs,
	}
}

// collectEvolutions fetches each unique chain once and flattens it to directed
// (from -> to) species-id edges.
func (s *Seeder) collectEvolutions(
	ctx context.Context, chainURLs map[string]struct{}, nameToID map[string]int, concurrency int,
) ([]pokemon.SeedEvolution, error) {
	urls := make([]string, 0, len(chainURLs))
	for u := range chainURLs {
		urls = append(urls, u)
	}
	sort.Strings(urls) // deterministic order

	var (
		mu    sync.Mutex
		edges []pokemon.SeedEvolution
	)
	err := parallel(ctx, urls, concurrency, func(ctx context.Context, url string) error {
		chain, err := s.client.GetEvolutionChain(ctx, url)
		if err != nil {
			return err
		}
		local := s.walkChain(chain.Chain, nameToID)

		mu.Lock()
		edges = append(edges, local...)
		mu.Unlock()
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].FromSpeciesID != edges[j].FromSpeciesID {
			return edges[i].FromSpeciesID < edges[j].FromSpeciesID
		}
		return edges[i].ToSpeciesID < edges[j].ToSpeciesID
	})
	return edges, nil
}

// walkChain turns a recursive evolution tree into flat (from -> to) edges,
// skipping any species not in nameToID (i.e. outside the imported set).
func (s *Seeder) walkChain(node pokeapi.ChainLink, nameToID map[string]int) []pokemon.SeedEvolution {
	var edges []pokemon.SeedEvolution
	fromID, fromOK := nameToID[node.Species.Name]
	for _, child := range node.EvolvesTo {
		toID, toOK := nameToID[child.Species.Name]
		if fromOK && toOK {
			edges = append(edges, pokemon.SeedEvolution{FromSpeciesID: fromID, ToSpeciesID: toID})
		}
		edges = append(edges, s.walkChain(child, nameToID)...)
	}
	return edges
}

// --- pure mapping helpers ---

func displayName(species *pokeapi.Species) string {
	for _, n := range species.Names {
		if n.Language.Name == "en" {
			return n.Name
		}
	}
	return species.Name
}

func nationalDexNumber(species *pokeapi.Species) int {
	for _, p := range species.PokedexNumbers {
		if p.Pokedex.Name == "national" {
			return p.EntryNumber
		}
	}
	return species.ID
}

func englishDescription(species *pokeapi.Species) *string {
	for _, e := range species.FlavorTextEntries {
		if e.Language.Name == "en" {
			cleaned := cleanFlavorText(e.FlavorText)
			return &cleaned
		}
	}
	return nil
}

// cleanFlavorText collapses PokéAPI's embedded newlines/form-feeds/soft-hyphens
// into single spaces.
func cleanFlavorText(s string) string {
	replaced := strings.NewReplacer("\n", " ", "\f", " ", "\r", " ", "\u00ad", "").Replace(s)
	return strings.Join(strings.Fields(replaced), " ")
}

// parseTrailingID extracts the numeric id from a PokéAPI resource URL such as
// ".../generation/1/". It returns 0 when no id is present.
func parseTrailingID(url string) int {
	trimmed := strings.Trim(url, "/")
	if trimmed == "" {
		return 0
	}
	parts := strings.Split(trimmed, "/")
	id, _ := strconv.Atoi(parts[len(parts)-1])
	return id
}

func positiveOrNil(v int) *int {
	if v <= 0 {
		return nil
	}
	return &v
}
