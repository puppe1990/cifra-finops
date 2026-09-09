-- up
ALTER TABLE cloud_accounts ADD COLUMN provider TEXT NOT NULL DEFAULT 'aws';

-- down
-- SQLite cannot drop a column in older versions; leave provider in place on rollback.
