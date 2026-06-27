-- Reference data only: generations and types. No species/forms seeded yet.

INSERT INTO pokemon_generations (id, name, region_name) VALUES
    (1, 'Generation I',   'Kanto'),
    (2, 'Generation II',  'Johto'),
    (3, 'Generation III', 'Hoenn'),
    (4, 'Generation IV',  'Sinnoh'),
    (5, 'Generation V',   'Unova'),
    (6, 'Generation VI',  'Kalos'),
    (7, 'Generation VII', 'Alola'),
    (8, 'Generation VIII','Galar'),
    (9, 'Generation IX',  'Paldea');

INSERT INTO pokemon_types (id, name, slug) VALUES
    (1,  'Normal',   'normal'),
    (2,  'Fire',     'fire'),
    (3,  'Water',    'water'),
    (4,  'Electric', 'electric'),
    (5,  'Grass',    'grass'),
    (6,  'Ice',      'ice'),
    (7,  'Fighting', 'fighting'),
    (8,  'Poison',   'poison'),
    (9,  'Ground',   'ground'),
    (10, 'Flying',   'flying'),
    (11, 'Psychic',  'psychic'),
    (12, 'Bug',      'bug'),
    (13, 'Rock',     'rock'),
    (14, 'Ghost',    'ghost'),
    (15, 'Dragon',   'dragon'),
    (16, 'Dark',     'dark'),
    (17, 'Steel',    'steel'),
    (18, 'Fairy',    'fairy');
