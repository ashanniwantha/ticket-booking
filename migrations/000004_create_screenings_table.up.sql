CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS screenings (
    id BIGSERIAL,
    movie_id BIGINT NOT NULL, 
    hall_id BIGINT NOT NULL,  
    screening_period TSTZRANGE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_screenings PRIMARY KEY (id),
    
    CONSTRAINT fk_screenings_movies FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE,
    CONSTRAINT fk_screenings_halls FOREIGN KEY (hall_id) REFERENCES halls(id) ON DELETE RESTRICT,

    -- FIXED: Extracted the lower bound timestamp to compare against NOW()
    CONSTRAINT chk_screening_period_future CHECK (lower(screening_period) > NOW()), 
    
    CONSTRAINT excl_screening_overlap EXCLUDE USING GIST (
        hall_id WITH =,
        screening_period WITH &&
    )
);

CREATE INDEX IF NOT EXISTS idx_screenings_movie_id ON screenings (movie_id); 

DROP TRIGGER IF EXISTS trg_screenings_set_updated_at ON screenings;

CREATE TRIGGER trg_screenings_set_updated_at
BEFORE UPDATE ON screenings
FOR EACH ROW 
EXECUTE FUNCTION set_updated_at();