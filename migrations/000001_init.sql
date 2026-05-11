CREATE TABLE IF NOT EXISTS raw_locations (
    code varchar(13) PRIMARY KEY,
    name varchar(100) NOT NULL,
    level smallint NOT NULL,
    imported_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS provinces (
    code varchar(2) PRIMARY KEY,
    name varchar(100) NOT NULL,
    source_code varchar(13) NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS regencies (
    code varchar(5) PRIMARY KEY,
    short_code varchar(2) NOT NULL,
    province_code varchar(2) NOT NULL REFERENCES provinces(code) ON DELETE CASCADE,
    name varchar(100) NOT NULL,
    source_code varchar(13) NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (province_code, short_code)
);

CREATE TABLE IF NOT EXISTS districts (
    code varchar(8) PRIMARY KEY,
    short_code varchar(2) NOT NULL,
    province_code varchar(2) NOT NULL REFERENCES provinces(code) ON DELETE CASCADE,
    regency_code varchar(5) NOT NULL REFERENCES regencies(code) ON DELETE CASCADE,
    name varchar(100) NOT NULL,
    source_code varchar(13) NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (regency_code, short_code)
);

CREATE TABLE IF NOT EXISTS villages (
    code varchar(13) PRIMARY KEY,
    short_code varchar(4) NOT NULL,
    province_code varchar(2) NOT NULL REFERENCES provinces(code) ON DELETE CASCADE,
    regency_code varchar(5) NOT NULL REFERENCES regencies(code) ON DELETE CASCADE,
    district_code varchar(8) NOT NULL REFERENCES districts(code) ON DELETE CASCADE,
    name varchar(100) NOT NULL,
    source_code varchar(13) NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (district_code, short_code)
);

CREATE INDEX IF NOT EXISTS idx_raw_locations_name ON raw_locations (name);
CREATE INDEX IF NOT EXISTS idx_regencies_province ON regencies (province_code, name);
CREATE INDEX IF NOT EXISTS idx_districts_regency ON districts (regency_code, name);
CREATE INDEX IF NOT EXISTS idx_villages_district ON villages (district_code, name);
CREATE INDEX IF NOT EXISTS idx_provinces_name ON provinces (name);
CREATE INDEX IF NOT EXISTS idx_regencies_name ON regencies (name);
CREATE INDEX IF NOT EXISTS idx_districts_name ON districts (name);
CREATE INDEX IF NOT EXISTS idx_villages_name ON villages (name);
