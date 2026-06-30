ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS organization_id uuid REFERENCES organizations(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS projects_organization_idx ON projects (organization_id);
