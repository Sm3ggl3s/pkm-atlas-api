-- Detail fields for the read-only Pokédex API.
--
-- height/weight live on FORMS (physical attributes can differ per form) and
-- description on SPECIES (Pokédex flavour text is species-level). All three are
-- nullable: the columns are added now and populated later (no data is seeded
-- here yet), so the detail endpoint returns null until that data exists.

ALTER TABLE pokemon_species
    ADD COLUMN description TEXT;

ALTER TABLE pokemon_forms
    ADD COLUMN height INTEGER CHECK (height IS NULL OR height > 0), -- decimetres (PokeAPI convention)
    ADD COLUMN weight INTEGER CHECK (weight IS NULL OR weight > 0); -- hectograms (PokeAPI convention)

-- Directed evolution edges: from_species evolves TO to_species. Seeded later;
-- the detail endpoint's "evolvesTo" is an empty list until then.
CREATE TABLE pokemon_evolutions (
    from_species_id INTEGER NOT NULL REFERENCES pokemon_species(id) ON DELETE CASCADE,
    to_species_id   INTEGER NOT NULL REFERENCES pokemon_species(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (from_species_id, to_species_id),
    CONSTRAINT pokemon_evolutions_no_self CHECK (from_species_id <> to_species_id)
);

CREATE INDEX idx_pokemon_evolutions_from ON pokemon_evolutions(from_species_id);
