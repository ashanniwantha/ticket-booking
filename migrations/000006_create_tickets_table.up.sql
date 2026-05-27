CREATE TABLE IF NOT EXISTS tickets (
    id BIGSERIAL,
    screening_id BIGINT NOT NULL,
    seat_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    status VARCHAR(12) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_tickets PRIMARY KEY (id),
    CONSTRAINT fk_tickets_screenings FOREIGN KEY (screening_id) REFERENCES screenings(id) ON DELETE CASCADE,
    CONSTRAINT fk_tickets_seats FOREIGN KEY (seat_id) REFERENCES seats(id) ON DELETE RESTRICT,
    CONSTRAINT fk_tickets_users FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,

    CONSTRAINT chk_tickets_status CHECK (status IN ('hold', 'booked', 'cancelled'))
);

CREATE UNIQUE INDEX uq_active_tickets 
ON tickets (screening_id, seat_id) 
WHERE status != 'cancelled';

DROP TRIGGER IF EXISTS trg_tickets_set_updated_at ON tickets;
CREATE TRIGGER trg_tickets_set_updated_at
BEFORE UPDATE ON tickets
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();