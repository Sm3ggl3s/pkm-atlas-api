-- Pokémon catalogue foundation.
--
-- Modelling note: types belong to FORMS, not species. Different forms of the
-- same species can have different typings (e.g. Mega Charizard X is Fire/Dragon
-- while the default Charizard is Fire/Flying), so typing is reached via
-- pokemon_forms -> pokemon_form_types -> pokemon_types.

-- Reusable trigger to keep updated_at current on UPDATE.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Generations and their regions.
CREATE TABLE pokemon_generations (
    id          INTEGER PRIMARY KEY, -- actual generation number (1, 2, 3, ...)
    name        TEXT NOT NULL UNIQUE CHECK (length(trim(name)) > 0),
    region_name TEXT NOT NULL CHECK (length(trim(region_name)) > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Main National Dex species. Carries no type columns by design.
CREATE TABLE pokemon_species (
    id                  INTEGER PRIMARY KEY,
    national_dex_number INTEGER NOT NULL UNIQUE CHECK (national_dex_number > 0),
    name                TEXT NOT NULL UNIQUE CHECK (length(trim(name)) > 0),
    slug                TEXT NOT NULL UNIQUE CHECK (length(trim(slug)) > 0),
    generation_id       INTEGER NOT NULL REFERENCES pokemon_generations(id),
    is_legendary        BOOLEAN NOT NULL DEFAULT FALSE,
    is_mythical         BOOLEAN NOT NULL DEFAULT FALSE,
    is_baby             BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Individual forms / variants of a species.
CREATE TABLE pokemon_forms (
    id             BIGSERIAL PRIMARY KEY,
    species_id     INTEGER NOT NULL REFERENCES pokemon_species(id) ON DELETE CASCADE,
    name           TEXT NOT NULL CHECK (length(trim(name)) > 0),
    slug           TEXT NOT NULL UNIQUE CHECK (length(trim(slug)) > 0),
    form_name      TEXT, -- optional short label (e.g. "Mega X", "Alolan")
    is_default     BOOLEAN NOT NULL DEFAULT FALSE,
    is_regional    BOOLEAN NOT NULL DEFAULT FALSE,
    is_mega        BOOLEAN NOT NULL DEFAULT FALSE,
    is_gigantamax  BOOLEAN NOT NULL DEFAULT FALSE,
    is_battle_only BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pokemon_forms_species_name_unique UNIQUE (species_id, name)
);

-- At most one default form per species (partial unique index — a plain UNIQUE
-- cannot express "only when is_default = TRUE").
CREATE UNIQUE INDEX idx_pokemon_forms_one_default_per_species
    ON pokemon_forms(species_id)
    WHERE is_default = TRUE;

-- The 18 Pokémon types (reference table).
CREATE TABLE pokemon_types (
    id         SMALLINT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE CHECK (length(trim(name)) > 0),
    slug       TEXT NOT NULL UNIQUE CHECK (length(trim(slug)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Join table giving each form one or two types, preserving slot order.
CREATE TABLE pokemon_form_types (
    form_id BIGINT NOT NULL REFERENCES pokemon_forms(id) ON DELETE CASCADE,
    type_id SMALLINT NOT NULL REFERENCES pokemon_types(id),
    slot    SMALLINT NOT NULL, -- 1 = primary, 2 = secondary

    PRIMARY KEY (form_id, type_id),
    CONSTRAINT pokemon_form_types_form_slot_unique UNIQUE (form_id, slot),
    CONSTRAINT pokemon_form_types_slot_check CHECK (slot IN (1, 2))
);

-- Lookup indexes.
CREATE INDEX idx_pokemon_species_generation_id ON pokemon_species(generation_id);
CREATE INDEX idx_pokemon_species_national_dex_number ON pokemon_species(national_dex_number);
CREATE INDEX idx_pokemon_forms_species_id ON pokemon_forms(species_id);
CREATE INDEX idx_pokemon_form_types_form_id ON pokemon_form_types(form_id);
CREATE INDEX idx_pokemon_form_types_type_id ON pokemon_form_types(type_id);

-- Keep updated_at fresh on every UPDATE. (pokemon_form_types has no updated_at.)
CREATE TRIGGER trg_pokemon_generations_set_updated_at
    BEFORE UPDATE ON pokemon_generations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_pokemon_species_set_updated_at
    BEFORE UPDATE ON pokemon_species
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_pokemon_forms_set_updated_at
    BEFORE UPDATE ON pokemon_forms
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_pokemon_types_set_updated_at
    BEFORE UPDATE ON pokemon_types
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
