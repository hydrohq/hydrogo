ALTER TABLE form_inbox ADD COLUMN location_country_code TEXT NOT NULL DEFAULT '';
ALTER TABLE form_inbox ADD COLUMN location_city TEXT NOT NULL DEFAULT '';
ALTER TABLE form_inbox ADD COLUMN location_local INTEGER NOT NULL DEFAULT 0 CHECK(location_local IN (0,1));
