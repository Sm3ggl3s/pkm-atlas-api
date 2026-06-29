-- Reverse of 000003.
DROP TABLE IF EXISTS pokemon_evolutions;

ALTER TABLE pokemon_forms
    DROP COLUMN IF EXISTS weight,
    DROP COLUMN IF EXISTS height;

ALTER TABLE pokemon_species
    DROP COLUMN IF EXISTS description;
