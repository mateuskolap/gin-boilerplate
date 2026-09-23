-- Create the initial application schema.
CREATE TABLE "users" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "deleted_at" timestamptz NULL,
  "name" varchar(100) NOT NULL,
  "email" varchar(320) NOT NULL,
  "password" varchar(255) NOT NULL,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX "idx_users_email_active" ON "users" (LOWER("email")) WHERE "deleted_at" IS NULL;
CREATE INDEX "idx_users_deleted_at" ON "users" ("deleted_at");

CREATE TABLE "roles" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "name" varchar(100) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "roles_name_key" UNIQUE ("name")
);

CREATE TABLE "permissions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "name" varchar(100) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "permissions_name_key" UNIQUE ("name")
);

CREATE TABLE "user_roles" (
  "user_id" uuid NOT NULL,
  "role_id" uuid NOT NULL,
  PRIMARY KEY ("user_id", "role_id"),
  CONSTRAINT "user_roles_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE,
  CONSTRAINT "user_roles_role_id_fkey" FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON DELETE CASCADE
);

CREATE INDEX "idx_user_roles_role_id" ON "user_roles" ("role_id");

CREATE TABLE "role_permissions" (
  "role_id" uuid NOT NULL,
  "permission_id" uuid NOT NULL,
  PRIMARY KEY ("role_id", "permission_id"),
  CONSTRAINT "role_permissions_role_id_fkey" FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON DELETE CASCADE,
  CONSTRAINT "role_permissions_permission_id_fkey" FOREIGN KEY ("permission_id") REFERENCES "permissions" ("id") ON DELETE CASCADE
);

CREATE INDEX "idx_role_permissions_permission_id" ON "role_permissions" ("permission_id");

CREATE TABLE "refresh_tokens" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "user_id" uuid NOT NULL,
  "token_hash" varchar(64) NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "revoked_at" timestamptz NULL,
  "replaced_by" uuid NULL,
  "ip_address" varchar(45) NOT NULL,
  "user_agent" text NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "refresh_tokens_token_hash_key" UNIQUE ("token_hash"),
  CONSTRAINT "refresh_tokens_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE
);

CREATE INDEX "idx_refresh_tokens_user_id" ON "refresh_tokens" ("user_id");
CREATE INDEX "idx_refresh_tokens_expires_at" ON "refresh_tokens" ("expires_at");
CREATE INDEX "idx_refresh_tokens_active_user" ON "refresh_tokens" ("user_id", "expires_at") WHERE "revoked_at" IS NULL;
