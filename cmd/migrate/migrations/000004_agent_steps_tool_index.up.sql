-- Supports ToolStats: the per-tool lifetime min(timestamp) ("is_new") and the
-- per-tool windowed aggregates. Without this, those scan agent_steps by
-- (project_id, timestamp) and filter tool_name row-by-row. Partial on
-- tool_name IS NOT NULL since all tool queries exclude null tool names.
CREATE INDEX IF NOT EXISTS agent_steps_project_tool_time_idx
    ON agent_steps (project_id, tool_name, timestamp)
    WHERE tool_name IS NOT NULL;
