CREATE TABLE IF NOT EXISTS seats (
    id BIGSERIAL,
    hall_id BIGINT NOT NULL,
    seat_number VARCHAR(8) NOT NULL,
    class VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_seats PRIMARY KEY (id),
    CONSTRAINT fk_seats_halls FOREIGN KEY (hall_id) REFERENCES halls(id) ON DELETE CASCADE,
    
    CONSTRAINT uq_seat_number UNIQUE (hall_id, seat_number),
    CONSTRAINT chk_seat_number_not_empty CHECK (length(trim(seat_number)) > 0),
    CONSTRAINT chk_class_not_empty CHECK (length(trim(class)) > 0),
    CONSTRAINT chk_class_valid CHECK (class IN ('vip', 'balcony', 'regular'))
);

CREATE INDEX IF NOT EXISTS idx_seats_class ON seats(class);

DROP TRIGGER IF EXISTS trg_seats_set_updated_at ON seats;

CREATE TRIGGER trg_seats_set_updated_at
BEFORE UPDATE ON seats
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();