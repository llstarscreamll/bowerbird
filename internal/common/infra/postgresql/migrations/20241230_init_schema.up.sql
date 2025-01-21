CREATE TABLE IF NOT EXISTS "public"."users" (
    "id" VARCHAR(26) PRIMARY KEY,
    "first_name" VARCHAR(255) NOT NULL,
    "last_name" VARCHAR(255) NOT NULL,
    "email" VARCHAR(255) NOT NULL,
    "photo_url" VARCHAR(255) NOT NULL,
    "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    unique("email")
);

CREATE TABLE IF NOT EXISTS "public"."sessions" (
    "id" VARCHAR(26) PRIMARY KEY,
    "user_id" VARCHAR(26) NOt NULL,
    "expires_at" TIMESTAMP NOt NULL,
    "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS "public"."mail_credentials" (
    "id" VARCHAR(26) PRIMARY KEY,
    "user_id" VARCHAR(26) NOt NULL,
    "mail_provider" VARCHAR(50) NOt NULL,
    "access_token" VARCHAR(255) NOt NULL,
    "refresh_token" VARCHAR(255) NOt NULL,
    "expires_at" TIMESTAMP NOt NULL,
    "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
