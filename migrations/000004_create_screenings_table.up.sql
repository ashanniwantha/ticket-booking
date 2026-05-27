CREATE TABLE IF NOT EXISTS screenings (
    id BIGSERIAL,
    movie_id BIGINT NOT NULL, 
    hall_id BIGINT NOT NULL,  
    screening_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_screenings PRIMARY KEY (id),
    
    CONSTRAINT fk_screenings_movies FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE,
    CONSTRAINT fk_screenings_halls FOREIGN KEY (hall_id) REFERENCES halls(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_screenings_movie_id ON screenings (movie_id); 
CREATE INDEX IF NOT EXISTS idx_screenings_hall_id ON screenings (hall_id); 

DROP TRIGGER IF EXISTS trg_screenings_set_updated_at ON screenings;

CREATE TRIGGER trg_screenings_set_updated_at
BEFORE UPDATE ON screenings
FOR EACH ROW 
EXECUTE FUNCTION set_updated_at();
