// Package pokeapi is a small read-only client for the public PokéAPI
// (https://pokeapi.co/api/v2). It only fetches and decodes the resources the
// seeder needs; all mapping into our own models lives in the seed package.
package pokeapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// NamedRef is PokéAPI's ubiquitous {name, url} reference object.
type NamedRef struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Species mirrors the fields of /pokemon-species/{id} that we seed from.
type Species struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	IsLegendary bool   `json:"is_legendary"`
	IsMythical  bool   `json:"is_mythical"`
	IsBaby      bool   `json:"is_baby"`
	Names       []struct {
		Language NamedRef `json:"language"`
		Name     string   `json:"name"`
	} `json:"names"`
	Generation        NamedRef `json:"generation"`
	FlavorTextEntries []struct {
		FlavorText string   `json:"flavor_text"`
		Language   NamedRef `json:"language"`
		Version    NamedRef `json:"version"`
	} `json:"flavor_text_entries"`
	EvolutionChain struct {
		URL string `json:"url"`
	} `json:"evolution_chain"`
	Varieties []struct {
		IsDefault bool     `json:"is_default"`
		Pokemon   NamedRef `json:"pokemon"`
	} `json:"varieties"`
	PokedexNumbers []struct {
		EntryNumber int      `json:"entry_number"`
		Pokedex     NamedRef `json:"pokedex"`
	} `json:"pokedex_numbers"`
}

// Pokemon mirrors the fields of /pokemon/{id} that we seed from. A species can
// have several of these ("varieties"/forms); height is in decimetres and weight
// in hectograms, matching PokéAPI.
type Pokemon struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Height    int    `json:"height"`
	Weight    int    `json:"weight"`
	IsDefault bool   `json:"is_default"`
	Types     []struct {
		Slot int      `json:"slot"`
		Type NamedRef `json:"type"`
	} `json:"types"`
}

// EvolutionChain mirrors /evolution-chain/{id}. Chain is a recursive tree.
type EvolutionChain struct {
	ID    int       `json:"id"`
	Chain ChainLink `json:"chain"`
}

// ChainLink is one node of the evolution tree.
type ChainLink struct {
	Species   NamedRef    `json:"species"`
	EvolvesTo []ChainLink `json:"evolves_to"`
}

// Client talks to PokéAPI over HTTP using only the standard library.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a Client rooted at baseURL (e.g. https://pokeapi.co/api/v2).
// A nil httpClient gets a sensible default with a timeout.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: httpClient}
}

// ListSpeciesRefs returns a reference to every species in the National Dex,
// ordered by PokéAPI's default ordering (national dex order).
func (c *Client) ListSpeciesRefs(ctx context.Context) ([]NamedRef, error) {
	var out struct {
		Results []NamedRef `json:"results"`
	}
	if err := c.getJSON(ctx, c.baseURL+"/pokemon-species?limit=100000", &out); err != nil {
		return nil, err
	}
	return out.Results, nil
}

// GetSpecies fetches a single species by its absolute resource URL.
func (c *Client) GetSpecies(ctx context.Context, url string) (*Species, error) {
	var s Species
	if err := c.getJSON(ctx, url, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// GetPokemon fetches a single pokemon (form/variety) by its absolute URL.
func (c *Client) GetPokemon(ctx context.Context, url string) (*Pokemon, error) {
	var p Pokemon
	if err := c.getJSON(ctx, url, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetEvolutionChain fetches a single evolution chain by its absolute URL.
func (c *Client) GetEvolutionChain(ctx context.Context, url string) (*EvolutionChain, error) {
	var e EvolutionChain
	if err := c.getJSON(ctx, url, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// getJSON performs a GET and decodes a 200 body into dst. It retries a few times
// on 429/5xx and transport errors with a small linear backoff; other non-2xx
// statuses (e.g. 404) fail immediately.
func (c *Client) getJSON(ctx context.Context, url string, dst any) error {
	const maxAttempts = 4

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("build request %s: %w", url, err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("get %s: %w", url, err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			err := json.NewDecoder(resp.Body).Decode(dst)
			_ = resp.Body.Close()
			if err != nil {
				return fmt.Errorf("decode %s: %w", url, err)
			}
			return nil
		}

		retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		_ = resp.Body.Close()
		lastErr = fmt.Errorf("get %s: unexpected status %d", url, resp.StatusCode)
		if !retryable {
			return lastErr
		}
	}
	return lastErr
}
