CREATE TABLE IF NOT EXISTS location_areas (
    code varchar(13) PRIMARY KEY
        REFERENCES raw_locations(code) ON DELETE CASCADE,
    area_km2 double precision NOT NULL,
    source varchar(255) NOT NULL,
    reference_date date NOT NULL,
    imported_at timestamptz NOT NULL DEFAULT now(),
    CHECK (
        area_km2 >= 0
        AND area_km2 <> 'NaN'::double precision
        AND area_km2 <> 'Infinity'::double precision
        AND area_km2 <> '-Infinity'::double precision
    )
);
