-- Initial schema: multi-tenant root (projects) and machine credentials (api_keys).
-- All future ingest tables FK to projects(id) and scope by project_id from the
-- API key lookup. uuidv7() is a Postgres 18 built-in; gives time-ordered UUIDs
-- so PKs cluster nicely on insert.
-- TimescaleDB extension is loaded here once; later migrations will create the
-- hypertables (ai_traces, api_events, infra_metrics, agent_runs, agent_steps).
CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS projects (
    id          uuid PRIMARY KEY DEFAULT uuidv7(),
    name        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id            uuid PRIMARY KEY DEFAULT uuidv7(),
    project_id    uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key_hash      bytea NOT NULL UNIQUE,
    name          text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_used_at  timestamptz
);

CREATE INDEX IF NOT EXISTS api_keys_project_id_idx ON api_keys(project_id);
