CREATE INDEX idx_jobs_dead_letter_created ON jobs (dead_lettered, created_at, id);
