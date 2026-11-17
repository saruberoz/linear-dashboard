-- Copy of the Linear workspace. Every table is rewritten by each successful sync
-- (fetch everything, derive, then write in one transaction); rows not seen are deleted.

CREATE TABLE teams (
    id              STRING PRIMARY KEY,
    key             STRING NOT NULL,
    name            STRING NOT NULL,
    estimation_type STRING NOT NULL -- Linear issueEstimationType; "notUsed" means estimates are off
);

CREATE TABLE users (
    id           STRING PRIMARY KEY,
    name         STRING NOT NULL,
    display_name STRING NOT NULL,
    active       BOOL   NOT NULL
);

CREATE TABLE labels (
    id         STRING PRIMARY KEY,
    kind       STRING NOT NULL CHECK (kind IN ('issue', 'project')),
    name       STRING NOT NULL,
    group_name STRING NULL
);

CREATE TABLE initiatives (
    id         STRING      PRIMARY KEY,
    name       STRING      NOT NULL,
    status     STRING      NOT NULL,
    url        STRING      NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    is_visible BOOL        NOT NULL -- name starts with [W] or [WS]
);

CREATE TABLE projects (
    id         STRING      PRIMARY KEY,
    name       STRING      NOT NULL,
    state      STRING      NOT NULL,
    url        STRING      NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    segments   STRING[]    NOT NULL DEFAULT ARRAY[] -- derived from project labels, canonical order
);

CREATE TABLE project_labels (
    project_id STRING NOT NULL,
    label_id   STRING NOT NULL,
    PRIMARY KEY (project_id, label_id)
);

CREATE TABLE initiative_projects (
    initiative_id STRING NOT NULL,
    project_id    STRING NOT NULL,
    PRIMARY KEY (initiative_id, project_id)
);

CREATE TABLE issues (
    id                       STRING      PRIMARY KEY,
    identifier               STRING      NOT NULL,
    title                    STRING      NOT NULL,
    url                      STRING      NOT NULL,
    team_id                  STRING      NOT NULL,
    project_id               STRING      NULL,
    parent_id                STRING      NULL,
    assignee_id              STRING      NULL,
    estimate                 FLOAT8      NULL,
    priority                 INT2        NOT NULL,
    state_name               STRING      NOT NULL,
    state_type               STRING      NOT NULL,
    created_at               TIMESTAMPTZ NOT NULL,
    started_at               TIMESTAMPTZ NULL,
    completed_at             TIMESTAMPTZ NULL,
    canceled_at              TIMESTAMPTZ NULL,
    archived_at              TIMESTAMPTZ NULL,
    -- derived over the full task tree at sync time
    effective_pts            FLOAT8      NOT NULL DEFAULT 0,
    has_estimated_descendant BOOL        NOT NULL DEFAULT false,
    is_missing_estimate      BOOL        NOT NULL DEFAULT false,
    task_segments            STRING[]    NOT NULL DEFAULT ARRAY[],
    INDEX issues_project_idx (project_id),
    INDEX issues_parent_idx (parent_id)
);

CREATE TABLE issue_labels (
    issue_id STRING NOT NULL,
    label_id STRING NOT NULL,
    PRIMARY KEY (issue_id, label_id)
);

CREATE TABLE sync_runs (
    id          INT8        PRIMARY KEY DEFAULT unique_rowid(),
    trigger     STRING      NOT NULL CHECK (trigger IN ('startup', 'schedule', 'manual')),
    status      STRING      NOT NULL CHECK (status IN ('running', 'succeeded', 'failed')),
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ NULL,
    stats       JSONB       NULL,
    error       STRING      NULL,
    INDEX sync_runs_started_idx (started_at DESC)
);
