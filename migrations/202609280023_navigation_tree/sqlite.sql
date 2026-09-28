ALTER TABLE navigation_items ADD COLUMN parent_id TEXT NOT NULL DEFAULT '';
ALTER TABLE navigation_items ADD COLUMN target TEXT NOT NULL DEFAULT '_self' CHECK(target IN ('_self','_blank'));
