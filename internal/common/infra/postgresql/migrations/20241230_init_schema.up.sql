CREATE TABLE "public"."users" (
    "id" VARCHAR(26) PRIMARY KEY,
    "first_name" VARCHAR(255) NOT NULL,
    "last_name" VARCHAR(255) NOT NULL,
    "email" VARCHAR(255) NOT NULL,
    "photo_url" VARCHAR(255) NOT NULL,
    "last_login_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    unique("email")
);

CREATE TABLE "public"."sessions" (
    "id" VARCHAR(26) PRIMARY KEY,
    "user_id" VARCHAR(26) NOT NULL,
    "expires_at" TIMESTAMP WITH TIME ZONE NOT NULL,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE "public"."mail_credentials" (
    "id" VARCHAR(26) PRIMARY KEY,
    "user_id" VARCHAR(26) NOT NULL,
    "mail_provider" VARCHAR(50) NOT NULL,
    "mail_address" VARCHAR(255) NOT NULL,
    "access_token" TEXT NOT NULL,
    "refresh_token" TEXT NOT NULL,
    "expires_at" TIMESTAMP WITH TIME ZONE NOT NULL,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX "unique_mail_credentials_user_id_and_email" ON "public"."mail_credentials" ("user_id", "mail_address");