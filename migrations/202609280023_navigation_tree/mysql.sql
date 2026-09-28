ALTER TABLE navigation_items ADD COLUMN parent_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE navigation_items ADD COLUMN target VARCHAR(8) NOT NULL DEFAULT '_self';
