-- Added after commerce_customer_orders already shipped (202610231302), so
-- this is an ALTER, not a rewrite of that table's CREATE statement -- see
-- AGENTS.md rule 3, never edit an already-released migration.
--
-- One generic set of columns for whichever ShippingProvider created the
-- shipment (modules/commerce.ShippingProvider -- Yurtiçi Kargo first, Aras/
-- MNG/others later via the same interface): carrier names which one, the
-- other three are opaque to commerce itself and only ever set together by
-- Service.CreateShipment.
ALTER TABLE commerce_customer_orders ADD COLUMN shipping_carrier TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN shipping_tracking_number TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN shipping_tracking_url TEXT NOT NULL DEFAULT '';
ALTER TABLE commerce_customer_orders ADD COLUMN shipped_at TEXT;
