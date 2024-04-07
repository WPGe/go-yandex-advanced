CREATE TABLE IF NOT EXISTS metrics (
     id VARCHAR NOT NULL,
     type VARCHAR NOT NULL,
     delta BIGINT,
     value DOUBLE PRECISION,
     UNIQUE (id, type)
);