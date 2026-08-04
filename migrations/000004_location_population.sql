CREATE TABLE IF NOT EXISTS location_population (
    code varchar(13) PRIMARY KEY
        REFERENCES raw_locations(code) ON DELETE CASCADE,
    male bigint NOT NULL,
    female bigint NOT NULL,
    total bigint NOT NULL,
    source varchar(255) NOT NULL,
    reference_date date NOT NULL,
    imported_at timestamptz NOT NULL DEFAULT now(),
    CHECK (male >= 0),
    CHECK (female >= 0),
    CHECK (total >= 0),
    CHECK (male + female = total)
);
