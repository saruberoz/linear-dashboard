ALTER TABLE sync_runs DROP CONSTRAINT check_trigger;
ALTER TABLE sync_runs ADD CONSTRAINT check_sync_trigger CHECK (trigger IN ('startup', 'schedule', 'manual', 'refresh'));
