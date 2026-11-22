-- Removes the two plan rows that 202611080001_saas_billing_subscription seeded (hosted-platform pricing that does not belong in core).
-- Rows already referenced by an order or subscription are kept.
DELETE FROM plans WHERE id IN ('plan_monthly', 'plan_yearly')
  AND NOT EXISTS (SELECT 1 FROM subscriptions s WHERE s.plan_id = plans.id)
  AND NOT EXISTS (SELECT 1 FROM orders o WHERE o.plan_id = plans.id);
