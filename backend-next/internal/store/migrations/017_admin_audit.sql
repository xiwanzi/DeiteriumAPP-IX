CREATE INDEX ix_audit_next_time_sequence ON audit_events_next(created_at, sequence_id);
CREATE INDEX ix_audit_next_actor_time ON audit_events_next(actor_id, created_at, sequence_id);
CREATE INDEX ix_audit_next_action_time ON audit_events_next(action, created_at, sequence_id);
