-- Saved custom dashboards (shared: no owner, everyone can edit). Filters and
-- insight specs are stored as the protobuf JSON of InsightFilters / InsightSpec.
CREATE TABLE dashboards (
    id         STRING      PRIMARY KEY DEFAULT gen_random_uuid()::STRING,
    name       STRING      NOT NULL,
    filters    JSONB       NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE insights (
    id           STRING      PRIMARY KEY DEFAULT gen_random_uuid()::STRING,
    dashboard_id STRING      NOT NULL REFERENCES dashboards (id) ON DELETE CASCADE,
    title        STRING      NOT NULL,
    spec         JSONB       NOT NULL,
    position     INT4        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    INDEX insights_dashboard_idx (dashboard_id, position)
);
