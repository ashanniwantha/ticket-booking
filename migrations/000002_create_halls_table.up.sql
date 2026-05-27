CREATE TABLE IF NOT EXISTS halls (
    id BIGSERIAL,
    name VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_halls PRIMARY KEY (id),

    CONSTRAINT uq_halls_name UNIQUE (name),
    CONSTRAINT chk_halls_name_not_empty CHECK (length(trim(name)) > 0)
);

DROP TRIGGER IF EXISTS trg_halls_set_updated_at ON halls;
CREATE TRIGGER trg_halls_set_updated_at
BEFORE UPDATE ON halls
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
