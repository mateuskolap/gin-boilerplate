DROP INDEX idx_users_organization_id;

ALTER TABLE users
    DROP CONSTRAINT fk_users_organization,
    DROP COLUMN organization_id;
