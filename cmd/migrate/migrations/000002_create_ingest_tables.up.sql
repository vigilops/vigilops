-- AI / LLM call traces ----------------------------------------------------
CREATE TABLE IF NOT EXISTS ai_traces (
    id              uuid        NOT NULL DEFAULT uuidv7(),
    timestamp       timestamptz NOT NULL DEFAULT now(),
    project_id      uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    model           text        NOT NULL,
    provider        text,
    input_tokens    int,
    output_tokens   int,
    total_tokens    int,
    cost_usd        numeric(12, 6),
    latency_ms      int,
    status          text        NOT NULL,
    error_message   text,
    request_id      text,
    agent_run_id    uuid,
    metadata        jsonb,
    PRIMARY KEY (id, timestamp)
) WITH (
    tsdb.hypertable,
    tsdb.partition_column = 'timestamp',
    tsdb.segmentby        = 'project_id'
);

CREATE INDEX IF NOT EXISTS ai_traces_project_time_idx       ON ai_traces (project_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS ai_traces_project_model_time_idx ON ai_traces (project_id, model, timestamp DESC);
CREATE INDEX IF NOT EXISTS ai_traces_metadata_gin_idx       ON ai_traces USING GIN (metadata);

-- API / HTTP request events -----------------------------------------------
CREATE TABLE IF NOT EXISTS api_events (
    id                  uuid        NOT NULL DEFAULT uuidv7(),
    timestamp           timestamptz NOT NULL DEFAULT now(),
    project_id          uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    service             text        NOT NULL,
    method              text        NOT NULL,
    path                text        NOT NULL,
    status_code         int         NOT NULL,
    duration_ms         int         NOT NULL,
    request_size_bytes  int,
    response_size_bytes int,
    ip                  inet,
    user_agent          text,
    error               text,
    metadata            jsonb,
    PRIMARY KEY (id, timestamp)
) WITH (
    tsdb.hypertable,
    tsdb.partition_column = 'timestamp',
    tsdb.segmentby        = 'project_id'
);

CREATE INDEX IF NOT EXISTS api_events_project_time_idx              ON api_events (project_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS api_events_project_service_path_time_idx ON api_events (project_id, service, path, timestamp DESC);
CREATE INDEX IF NOT EXISTS api_events_project_status_time_idx       ON api_events (project_id, status_code, timestamp DESC);

-- Infrastructure metrics --------------------------------------------------
CREATE TABLE IF NOT EXISTS infra_metrics (
    timestamp   timestamptz      NOT NULL DEFAULT now(),
    project_id  uuid             NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    host        text             NOT NULL,
    metric_name text             NOT NULL,
    value       double precision NOT NULL,
    labels      jsonb,
    PRIMARY KEY (project_id, host, metric_name, timestamp)
) WITH (
    tsdb.hypertable,
    tsdb.partition_column = 'timestamp',
    tsdb.segmentby        = 'project_id, host'
);

CREATE INDEX IF NOT EXISTS infra_metrics_project_metric_time_idx ON infra_metrics (project_id, metric_name, timestamp DESC);
