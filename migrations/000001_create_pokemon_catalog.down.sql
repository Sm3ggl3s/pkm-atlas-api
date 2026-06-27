-- Reverse of 000001 in safe dependency order.

-- Drop triggers before the function they depend on.
DROP TRIGGER IF EXISTS trg_pokemon_types_set_updated_at ON pokemon_types;
DROP TRIGGER IF EXISTS trg_pokemon_forms_set_updated_at ON pokemon_forms;
DROP TRIGGER IF EXISTS trg_pokemon_species_set_updated_at ON pokemon_species;
DROP TRIGGER IF EXISTS trg_pokemon_generations_set_updated_at ON pokemon_generations;

-- Drop tables (indexes drop with their tables). Children before parents.
DROP TABLE IF EXISTS pokemon_form_types;
DROP TABLE IF EXISTS pokemon_types;
DROP TABLE IF EXISTS pokemon_forms;
DROP TABLE IF EXISTS pokemon_species;
DROP TABLE IF EXISTS pokemon_generations;

-- Drop the shared trigger function last.
DROP FUNCTION IF EXISTS set_updated_at();
