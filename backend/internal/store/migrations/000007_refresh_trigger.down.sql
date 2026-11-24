UPDATE sync_runs SET trigger = 'manual' WHERE trigger = 'refresh';
ALTER TABLE sync_runs DROP CONSTRAINT check_sync_trigger;
ALTER TABLE sync_runs ADD CONSTRAINT check_trigger CHECK (trigger IN ('startup', 'schedule', 'manual'));
