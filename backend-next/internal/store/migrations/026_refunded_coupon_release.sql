-- Restore only proven, fully refunded official orders. Coupon definitions,
-- order snapshots, receipt/batch history and all other reservations stay intact.
-- The order ID predicate prevents a replay from touching a later use of the coupon.
DELETE p FROM promotion_redemptions_v209 p
JOIN commerce_resources_v2 r ON r.resource_id=p.resource_id
WHERE r.channel='OFFICIAL_STORE' AND r.state='REFUNDED' AND r.funds_state='REFUNDED'
AND r.pending_operation_id IS NULL
AND CAST(r.settled_amount AS DECIMAL(20,2))=0
AND CAST(r.refunded_amount AS DECIMAL(20,2))=CAST(r.amount AS DECIMAL(20,2));
