ALTER TABLE users
    ADD COLUMN organization_id UUID,
    ADD CONSTRAINT fk_users_organization
        FOREIGN KEY (organization_id) REFERENCES organizations (id);

CREATE INDEX idx_users_organization_id ON users (organization_id);
