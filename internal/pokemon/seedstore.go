package pokemon

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedSpecies is the fully-resolved input for upserting one species and all of
// its forms. The seeder builds these from PokéAPI; all type ids are already
// resolved (see SeedForm.TypeIDs).
type SeedSpecies struct {
	ID                int
	NationalDexNumber int
	Name              string
	Slug              string
	GenerationID      int
	IsLegendary       bool
	IsMythical        bool
	IsBaby            bool
	Description       *string
	Forms             []SeedForm
}

// SeedForm is one form/variety of a species.
type SeedForm struct {
	Slug      string
	Name      string
	IsDefault bool
	Height    *int
	Weight    *int
	TypeIDs   []int // resolved type ids in slot order (slot = index+1)
}

// SeedEvolution is a directed "from evolves to" edge between two species ids.
type SeedEvolution struct {
	FromSpeciesID int
	ToSpeciesID   int
}

// SeedStore holds the write SQL used by the seeder. Keeping it beside the read
// Repository means all SQL for this domain lives in one package.
type SeedStore struct {
	pool *pgxpool.Pool
}

// NewSeedStore returns a SeedStore backed by the given pool.
func NewSeedStore(pool *pgxpool.Pool) *SeedStore {
	return &SeedStore{pool: pool}
}

// TypeIDsBySlug loads the slug -> id map for the reference types so the seeder
// can resolve a form's type slugs without a query per form.
func (s *SeedStore) TypeIDsBySlug(ctx context.Context) (map[string]int, error) {
	rows, err := s.pool.Query(ctx, `SELECT slug, id FROM pokemon_types`)
	if err != nil {
		return nil, fmt.Errorf("query types: %w", err)
	}
	defer rows.Close()

	out := make(map[string]int)
	for rows.Next() {
		var (
			slug string
			id   int
		)
		if err := rows.Scan(&slug, &id); err != nil {
			return nil, fmt.Errorf("scan type: %w", err)
		}
		out[slug] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate types: %w", err)
	}
	return out, nil
}

// UpsertSpecies writes one species and its forms (with their types) in a single
// transaction. It is idempotent: re-running refreshes existing rows.
func (s *SeedStore) UpsertSpecies(ctx context.Context, sp SeedSpecies) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
INSERT INTO pokemon_species
    (id, national_dex_number, name, slug, generation_id, is_legendary, is_mythical, is_baby, description)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET
    national_dex_number = EXCLUDED.national_dex_number,
    name                = EXCLUDED.name,
    slug                = EXCLUDED.slug,
    generation_id       = EXCLUDED.generation_id,
    is_legendary        = EXCLUDED.is_legendary,
    is_mythical         = EXCLUDED.is_mythical,
    is_baby             = EXCLUDED.is_baby,
    description         = EXCLUDED.description`,
		sp.ID, sp.NationalDexNumber, sp.Name, sp.Slug, sp.GenerationID,
		sp.IsLegendary, sp.IsMythical, sp.IsBaby, sp.Description); err != nil {
		return fmt.Errorf("upsert species %d: %w", sp.ID, err)
	}

	// Clear the default flag first so re-runs can move it between forms without
	// tripping the "one default form per species" partial unique index.
	if _, err := tx.Exec(ctx,
		`UPDATE pokemon_forms SET is_default = FALSE WHERE species_id = $1`, sp.ID); err != nil {
		return fmt.Errorf("reset default forms for species %d: %w", sp.ID, err)
	}

	for _, f := range sp.Forms {
		var formID int64
		if err := tx.QueryRow(ctx, `
INSERT INTO pokemon_forms (species_id, name, slug, is_default, height, weight)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (slug) DO UPDATE SET
    species_id = EXCLUDED.species_id,
    name       = EXCLUDED.name,
    is_default = EXCLUDED.is_default,
    height     = EXCLUDED.height,
    weight     = EXCLUDED.weight
RETURNING id`,
			sp.ID, f.Name, f.Slug, f.IsDefault, f.Height, f.Weight).Scan(&formID); err != nil {
			return fmt.Errorf("upsert form %q: %w", f.Slug, err)
		}

		if _, err := tx.Exec(ctx,
			`DELETE FROM pokemon_form_types WHERE form_id = $1`, formID); err != nil {
			return fmt.Errorf("clear form types for %q: %w", f.Slug, err)
		}
		for i, typeID := range f.TypeIDs {
			if _, err := tx.Exec(ctx,
				`INSERT INTO pokemon_form_types (form_id, type_id, slot) VALUES ($1, $2, $3)`,
				formID, typeID, i+1); err != nil {
				return fmt.Errorf("insert form type for %q: %w", f.Slug, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit species %d: %w", sp.ID, err)
	}
	return nil
}

// ReplaceEvolutions rebuilds the whole evolution edge set in one transaction, so
// the table stays a 1-1 mirror of the source on every run.
func (s *SeedStore) ReplaceEvolutions(ctx context.Context, edges []SeedEvolution) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `TRUNCATE pokemon_evolutions`); err != nil {
		return fmt.Errorf("truncate evolutions: %w", err)
	}

	batch := &pgx.Batch{}
	for _, e := range edges {
		batch.Queue(
			`INSERT INTO pokemon_evolutions (from_species_id, to_species_id)
             VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			e.FromSpeciesID, e.ToSpeciesID)
	}
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("insert evolutions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit evolutions: %w", err)
	}
	return nil
}
