CREATE TABLE IF NOT EXISTS location_boundaries (
    code varchar(13) PRIMARY KEY REFERENCES raw_locations(code) ON DELETE CASCADE,
    centroid_lat double precision NOT NULL,
    centroid_lng double precision NOT NULL,
    leaflet_path jsonb NOT NULL,
    imported_at timestamptz NOT NULL DEFAULT now(),
    CHECK (centroid_lat BETWEEN -90 AND 90),
    CHECK (centroid_lng BETWEEN -180 AND 180)
);
