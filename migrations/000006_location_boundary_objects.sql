ALTER TABLE location_boundaries
    ADD COLUMN IF NOT EXISTS object_key text;

ALTER TABLE location_boundaries
    ALTER COLUMN leaflet_path DROP NOT NULL;

CREATE INDEX IF NOT EXISTS location_boundaries_object_key_idx
    ON location_boundaries (object_key)
    WHERE object_key IS NOT NULL;
