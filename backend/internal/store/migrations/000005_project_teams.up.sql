CREATE TABLE project_teams (
    project_id STRING NOT NULL,
    team_id    STRING NOT NULL,
    PRIMARY KEY (project_id, team_id)
);
