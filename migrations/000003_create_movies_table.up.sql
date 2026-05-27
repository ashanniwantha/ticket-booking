CREATE TABLE IF NOT EXISTS movies (
    id BIGSERIAL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_movies PRIMARY KEY (id),
    
    CONSTRAINT chk_movies_title_not_empty CHECK (length(trim(title)) > 0)
);

DROP TRIGGER IF EXISTS trg_movies_set_updated_at ON movies;
CREATE TRIGGER trg_movies_set_updated_at
BEFORE UPDATE ON movies
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
