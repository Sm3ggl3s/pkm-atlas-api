// Package pokemon provides the read-only Pokédex API: response models, a
// Postgres repository (all SQL lives here), and thin HTTP handlers.
package pokemon

// ListItem is the summary shape returned by the list endpoints.
type ListItem struct {
	ID                int      `json:"id"`
	NationalDexNumber int      `json:"nationalDexNumber"`
	Name              string   `json:"name"`
	Slug              string   `json:"slug"`
	Types             []string `json:"types"`
}

// Detail is the full shape for a single Pokémon. Height, Weight and
// Description are nullable (pointers) — populated by the seeder, serialising as
// JSON null when the source has no value. EvolvesTo is always present, empty
// when there are no known evolutions.
type Detail struct {
	ID                int      `json:"id"`
	NationalDexNumber int      `json:"nationalDexNumber"`
	Name              string   `json:"name"`
	Slug              string   `json:"slug"`
	Types             []string `json:"types"`
	Height            *int     `json:"height"`
	Weight            *int     `json:"weight"`
	Description       *string  `json:"description"`
	EvolvesTo         []string `json:"evolvesTo"`
}

// TypeResponse is the shape returned by the types endpoint.
type TypeResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}
