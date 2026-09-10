ALTER TABLE cost_lines ADD COLUMN currency TEXT NOT NULL DEFAULT '';
ALTER TABLE cloud_resources ADD COLUMN currency TEXT NOT NULL DEFAULT '';

-- down
-- SQLite cannot drop a column in older versions; leave currency in place on rollback.
