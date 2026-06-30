
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS users (
    id          uuid PRIMARY KEY DEFAULT uuidv7(),
    email       citext NOT NULL UNIQUE,
    password    bytea,
    name        text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    is_verified bool        NOT NULL DEFAULT false,
    verified_at timestamptz
);

CREATE TABLE IF NOT EXISTS organizations (
    id          uuid PRIMARY KEY DEFAULT uuidv7(),
    name        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS organization_members (
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            text NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY     (organization_id, user_id)
);

CREATE INDEX IF NOT EXISTS organization_members_user_idx
    ON organization_members (user_id);

CREATE TABLE IF NOT EXISTS organization_invites (
    id              uuid PRIMARY KEY DEFAULT uuidv7(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email           citext NOT NULL,
    role            text NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    token           bytea NOT NULL UNIQUE,
    expires_at      timestamptz NOT NULL,
    accepted_at     timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS organization_invites_organization_idx
    ON organization_invites (organization_id);
CREATE INDEX IF NOT EXISTS organization_invites_email_idx
    ON organization_invites (email);

CREATE TABLE IF NOT EXISTS user_verifications (
    id          uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       bytea NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS user_verifications_user_idx ON user_verifications (user_id);

CREATE TABLE IF NOT EXISTS sessions (
    id            uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash    bytea NOT NULL UNIQUE,
    created_at    timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL,
    last_used_at  timestamptz
);

CREATE INDEX IF NOT EXISTS sessions_user_idx ON sessions (user_id);
CREATE INDEX IF NOT EXISTS sessions_expires_idx ON sessions (expires_at);

CREATE TABLE IF NOT EXISTS user_identities (
    id                uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider          text NOT NULL CHECK (provider IN ('password', 'google', 'github')),
    provider_user_id  text NOT NULL,
    email             citext NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_user_id)
);

CREATE INDEX IF NOT EXISTS user_identities_user_idx ON user_identities (user_id);
