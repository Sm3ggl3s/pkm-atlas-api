package pokemon

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a requested Pokémon or type does not exist.
var ErrNotFound = errors.New("not found")

// Repository reads Pokédex data from Postgres. All read SQL lives here so the
// handlers stay thin and free of database concerns.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository returns a Repository backed by the given pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// listSelect is the shared projection for the list shape: a species joined to
// its default form, with that form's type slugs aggregated in slot order.
const listSelect = `
SELECT s.id,
       s.national_dex_number,
       s.name,
       s.slug,
       COALESCE(array_agg(t.slug ORDER BY ft.slot) FILTER (WHERE t.slug IS NOT NULL), '{}') AS types
FROM pokemon_species s
JOIN pokemon_forms f ON f.species_id = s.id AND f.is_default
LEFT JOIN pokemon_form_types ft ON ft.form_id = f.id
LEFT JOIN pokemon_types t ON t.id = ft.type_id`

// ListPokemon returns every species in national dex order.
func (r *Repository) ListPokemon(ctx context.Context) ([]ListItem, error) {
	rows, err := r.pool.Query(ctx, listSelect+`
GROUP BY s.id, s.national_dex_number, s.name, s.slug
ORDER BY s.national_dex_number`)
	if err != nil {
		return nil, fmt.Errorf("query pokemon: %w", err)
	}
	defer rows.Close()
	return scanListItems(rows)
}

// ListPokemonByType returns every species whose default form has the given type,
// in national dex order. It returns ErrNotFound when the type slug is unknown.
func (r *Repository) ListPokemonByType(ctx context.Context, typeSlug string) ([]ListItem, error) {
	var typeID int
	err := r.pool.QueryRow(ctx, `SELECT id FROM pokemon_types WHERE slug = $1`, typeSlug).Scan(&typeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query type %q: %w", typeSlug, err)
	}

	rows, err := r.pool.Query(ctx, listSelect+`
WHERE f.id IN (SELECT form_id FROM pokemon_form_types WHERE type_id = $1)
GROUP BY s.id, s.national_dex_number, s.name, s.slug
ORDER BY s.national_dex_number`, typeID)
	if err != nil {
		return nil, fmt.Errorf("query pokemon by type %q: %w", typeSlug, err)
	}
	defer rows.Close()
	return scanListItems(rows)
}

// GetPokemonBySlug returns full detail for a single species, or ErrNotFound.
func (r *Repository) GetPokemonBySlug(ctx context.Context, slug string) (*Detail, error) {
	var d Detail
	err := r.pool.QueryRow(ctx, `
SELECT s.id,
       s.national_dex_number,
       s.name,
       s.slug,
       COALESCE(array_agg(t.slug ORDER BY ft.slot) FILTER (WHERE t.slug IS NOT NULL), '{}') AS types,
       f.height,
       f.weight,
       s.description
FROM pokemon_species s
JOIN pokemon_forms f ON f.species_id = s.id AND f.is_default
LEFT JOIN pokemon_form_types ft ON ft.form_id = f.id
LEFT JOIN pokemon_types t ON t.id = ft.type_id
WHERE s.slug = $1
GROUP BY s.id, s.national_dex_number, s.name, s.slug, f.height, f.weight, s.description`, slug).
		Scan(&d.ID, &d.NationalDexNumber, &d.Name, &d.Slug, &d.Types, &d.Height, &d.Weight, &d.Description)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query pokemon %q: %w", slug, err)
	}

	evo, err := r.evolvesTo(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	d.EvolvesTo = evo
	return &d, nil
}

// evolvesTo returns the slugs a species evolves into, in dex order (empty when
// there are none).
func (r *Repository) evolvesTo(ctx context.Context, speciesID int) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
SELECT child.slug
FROM pokemon_evolutions e
JOIN pokemon_species child ON child.id = e.to_species_id
WHERE e.from_species_id = $1
ORDER BY child.national_dex_number`, speciesID)
	if err != nil {
		return nil, fmt.Errorf("query evolutions: %w", err)
	}
	defer rows.Close()

	slugs := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan evolution: %w", err)
		}
		slugs = append(slugs, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evolutions: %w", err)
	}
	return slugs, nil
}

// ListTypes returns all Pokémon types ordered by id.
func (r *Repository) ListTypes(ctx context.Context) ([]TypeResponse, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, slug FROM pokemon_types ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query types: %w", err)
	}
	defer rows.Close()

	types := []TypeResponse{}
	for rows.Next() {
		var t TypeResponse
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug); err != nil {
			return nil, fmt.Errorf("scan type: %w", err)
		}
		types = append(types, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate types: %w", err)
	}
	return types, nil
}

// scanListItems collects ListItem rows produced by listSelect.
func scanListItems(rows pgx.Rows) ([]ListItem, error) {
	items := []ListItem{}
	for rows.Next() {
		var it ListItem
		if err := rows.Scan(&it.ID, &it.NationalDexNumber, &it.Name, &it.Slug, &it.Types); err != nil {
			return nil, fmt.Errorf("scan pokemon: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pokemon: %w", err)
	}
	return items, nil
}
