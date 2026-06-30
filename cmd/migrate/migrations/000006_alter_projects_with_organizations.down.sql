DROP INDEX IF EXISTS projects_organization_idx;
ALTER TABLE projects DROP COLUMN IF EXISTS organization_id;
