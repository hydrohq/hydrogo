CREATE TABLE IF NOT EXISTS job_queue_capacity_lock (
    id INTEGER PRIMARY KEY
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
INSERT INTO job_queue_capacity_lock (id) VALUES (1) ON DUPLICATE KEY UPDATE id = id;
