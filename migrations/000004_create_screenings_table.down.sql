DROP TRIGGER IF EXISTS trg_screenings_set_updated_at ON screenings;

DROP TABLE IF EXISTS screenings;

-- Drop the extension since we want a completely clean slate
DROP EXTENSION IF EXISTS btree_gist;