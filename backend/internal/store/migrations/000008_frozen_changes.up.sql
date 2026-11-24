CREATE TABLE frozen_changes (
    id            INT8        PRIMARY KEY DEFAULT unique_rowid(),
    change_key    STRING      NOT NULL,
    kind          STRING      NOT NULL CHECK (kind IN ('estimate', 'project', 'initiative')),
    task_id       STRING      NOT NULL DEFAULT '',
    project_id    STRING      NOT NULL DEFAULT '',
    initiative_id STRING      NOT NULL DEFAULT '',
    old_value     STRING      NOT NULL DEFAULT '',
    new_value     STRING      NOT NULL DEFAULT '',
    frozen_by     STRING      NOT NULL,
    status        STRING      NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'withdrawn')),
    detected_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_at    TIMESTAMPTZ,
    reason        STRING      NOT NULL DEFAULT '',
    UNIQUE INDEX frozen_changes_one_pending (change_key) WHERE status = 'pending',
    INDEX frozen_changes_by_status (status, detected_at DESC)
);
