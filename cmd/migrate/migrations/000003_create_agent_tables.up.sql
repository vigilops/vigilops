-- Agent runs --------------------------------------------------------------
CREATE TABLE IF NOT EXISTS agent_runs (
    id                  uuid        NOT NULL DEFAULT uuidv7(),
    timestamp           timestamptz NOT NULL DEFAULT now(),
    project_id          uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    agent_name          text        NOT NULL,
    status              text        NOT NULL,
    termination_reason  text,
    loop_detected       boolean     NOT NULL DEFAULT false,
    loop_step_index     int,
    total_steps         int         NOT NULL DEFAULT 0,
    total_tokens        int         NOT NULL DEFAULT 0,
    total_cost_usd      numeric(12, 6),
    duration_ms         int,
    input               text,
    output              text,
    metadata            jsonb,
    finished_at         timestamptz,
    PRIMARY KEY (id, timestamp)
) WITH (
    tsdb.hypertable,
    tsdb.partition_column = 'timestamp',
    tsdb.segmentby        = 'project_id'
);

CREATE INDEX IF NOT EXISTS agent_runs_project_time_idx        ON agent_runs (project_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS agent_runs_project_status_time_idx ON agent_runs (project_id, status, timestamp DESC);
CREATE INDEX IF NOT EXISTS agent_runs_project_agent_time_idx  ON agent_runs (project_id, agent_name, timestamp DESC);
CREATE INDEX IF NOT EXISTS agent_runs_project_loop_time_idx   ON agent_runs (project_id, loop_detected, timestamp DESC)
    WHERE loop_detected = true;

-- Agent steps -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS agent_steps (
    id                  uuid        NOT NULL DEFAULT uuidv7(),
    timestamp           timestamptz NOT NULL DEFAULT now(),
    project_id          uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    agent_run_id        uuid        NOT NULL,
    step_index          int         NOT NULL,
    step_type           text        NOT NULL,
    content             text,
    tool_name           text,
    tool_input          jsonb,
    tool_output         jsonb,
    tool_success        boolean,
    tool_latency_ms     int,
    input_fingerprint   bytea,
    tokens              int,
    cost_usd            numeric(12, 6),
    metadata            jsonb,
    PRIMARY KEY (id, timestamp)
) WITH (
    tsdb.hypertable,
    tsdb.partition_column = 'timestamp',
    tsdb.segmentby        = 'project_id, agent_run_id'
);

CREATE INDEX IF NOT EXISTS agent_steps_run_index_idx       ON agent_steps (agent_run_id, step_index);
CREATE INDEX IF NOT EXISTS agent_steps_project_time_idx    ON agent_steps (project_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS agent_steps_run_fingerprint_idx ON agent_steps (agent_run_id, input_fingerprint)
    WHERE input_fingerprint IS NOT NULL;
CREATE INDEX IF NOT EXISTS agent_steps_tool_input_gin_idx  ON agent_steps USING GIN (tool_input);

-- Agent tools registry ----------------------------------------------------
CREATE TABLE IF NOT EXISTS agent_tools (
    id              uuid        PRIMARY KEY DEFAULT uuidv7(),
    project_id      uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tool_name       text        NOT NULL,
    description     text,
    first_seen_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, tool_name)
);

CREATE INDEX IF NOT EXISTS agent_tools_project_last_seen_idx ON agent_tools (project_id, last_seen_at DESC);

-- Agent evaluations -------------------------------------------------------
CREATE TABLE IF NOT EXISTS agent_evaluations (
    id              uuid        PRIMARY KEY DEFAULT uuidv7(),
    project_id      uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    agent_run_id    uuid        NOT NULL,
    correctness     numeric(3, 2) CHECK (correctness  BETWEEN 0 AND 1),
    completeness    numeric(3, 2) CHECK (completeness BETWEEN 0 AND 1),
    efficiency      numeric(3, 2) CHECK (efficiency   BETWEEN 0 AND 1),
    safety          numeric(3, 2) CHECK (safety       BETWEEN 0 AND 1),
    evaluator       text        NOT NULL,
    evaluated_at    timestamptz NOT NULL DEFAULT now(),
    notes           text
);

CREATE INDEX IF NOT EXISTS agent_evaluations_run_idx          ON agent_evaluations (agent_run_id);
CREATE INDEX IF NOT EXISTS agent_evaluations_project_time_idx ON agent_evaluations (project_id, evaluated_at DESC);
