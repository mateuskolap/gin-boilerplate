CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    name VARCHAR(255) NOT NULL,
);

CREATE INDEX idx_organizations_deleted_at ON organizations (deleted_at);
CREATE UNIQUE INDEX idx_organizations_name_active ON organizations (name) WHERE deleted_at IS NULL;
