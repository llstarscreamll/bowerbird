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
    "id" VARCHAR(53) PRIMARY KEY,
    "user_id" VARCHAR(26) NOT NULL,
    "expires_at" TIMESTAMP WITH TIME ZONE NOT NULL,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE "public"."mail_credentials" (
    "id" VARCHAR(26) PRIMARY KEY,
    "user_id" VARCHAR(26) NOT NULL,
    "wallet_id" VARCHAR(26) NOT NULL,
    "mail_provider" VARCHAR(50) NOT NULL,
    "mail_address" VARCHAR(255) NOT NULL,
    "access_token" TEXT NOT NULL,
    "refresh_token" TEXT NOT NULL,
    "expires_at" TIMESTAMP WITH TIME ZONE NOT NULL,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX "unique_mail_credentials_wallet_id_and_email" ON "public"."mail_credentials" ("wallet_id", "mail_address");

CREATE TABLE "public"."mail_messages" (
    "id" VARCHAR(26) PRIMARY KEY,
    "external_id" VARCHAR(50) NOT NULL,
    "user_id" VARCHAR(26) NOT NULL,
    "from" VARCHAR(255) NOT NULL,
    "to" VARCHAR(255) NOT NULL,
    "subject" VARCHAR(255) NOT NULL,
    "body" TEXT NOT NULL,
    "received_at" TIMESTAMP WITH TIME ZONE NOT NULL,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX "unique_mail_messages_user_id_and_external_id" ON "public"."mail_messages" ("user_id", "external_id");

CREATE TABLE "public"."wallets" (
    "id" VARCHAR(26) PRIMARY KEY,
    "name" VARCHAR(255) NOT NULL,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE "public"."user_has_wallets" (
    "user_id" VARCHAR(26) NOT NULL,
    "wallet_id" VARCHAR(26) NOT NULL,
    "role" VARCHAR(10) NOT NULL,
    "joined_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE "public"."transactions" (
    "id" VARCHAR(26) PRIMARY KEY,
    "wallet_id" VARCHAR(26) NOT NULL,
    "user_id" VARCHAR(26) NOT NULL,
    "origin" VARCHAR(50) NOT NULL,
    "reference" VARCHAR(50) NOT NULL,
    "type" VARCHAR(50) NOT NULL,
    "amount" NUMERIC(10, 2) NOT NULL,
    "user_description" TEXT NOT NULL,
    "system_description" TEXT NOT NULL,
    "processed_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);