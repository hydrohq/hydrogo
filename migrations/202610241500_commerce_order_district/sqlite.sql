-- Added after commerce_customer_orders already shipped (202610231302), so
-- this is an ALTER, not a rewrite of that table's CREATE statement -- see
-- AGENTS.md rule 3, never edit an already-released migration.
--
-- Closes a real gap found while building the Yurtiçi Kargo shipping
-- provider (docs/decisions/0020): guest_city alone is not enough to ship
-- a package in Turkey -- Yurtiçi's schema (and every other Turkish
-- carrier) wants both a city (il) AND a district (ilçe). Before this
-- column existed, CreateShipment sent the same value for both.
ALTER TABLE commerce_customer_orders ADD COLUMN guest_district TEXT NOT NULL DEFAULT '';
