-- Remove only the rows seeded in 000002; leave the schema intact.

DELETE FROM pokemon_types WHERE id BETWEEN 1 AND 18;
DELETE FROM pokemon_generations WHERE id BETWEEN 1 AND 9;
