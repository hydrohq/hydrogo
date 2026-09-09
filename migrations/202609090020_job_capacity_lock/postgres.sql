CREATE TABLE IF NOT EXISTS job_queue_capacity_lock (
    id INTEGER PRIMARY KEY
);
INSERT INTO job_queue_capacity_lock (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
