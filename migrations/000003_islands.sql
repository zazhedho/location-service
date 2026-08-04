CREATE TABLE IF NOT EXISTS islands (
    code varchar(11) PRIMARY KEY,
    province_code varchar(2),
    name varchar(255) NOT NULL,
    latitude double precision,
    longitude double precision,
    status varchar(10),
    area double precision,
    notes text,
    imported_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_islands_province_name
    ON islands (province_code, name);
