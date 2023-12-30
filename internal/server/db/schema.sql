CREATE TABLE users (
        id INTEGER PRIMARY KEY,
        email TEXT NOT NULL UNIQUE,
        first_name TEXT NULL,
        last_name TEXT NULL,
        is_super_user BOOLEAN NOT NULL DEFAULT false,
        github_access_token TEXT NULL,
        github_avatar_url TEXT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
CREATE TABLE teams (
        id INTEGER PRIMARY KEY,
        NAME TEXT NOT NULL UNIQUE,
        slug TEXT NOT NULL UNIQUE,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
CREATE TABLE team_members (
        id INTEGER PRIMARY KEY,
        user_id INTEGER NOT NULL REFERENCES users (id),
        team_id INTEGER NOT NULL REFERENCES teams (id),
        secret_key TEXT NOT NULL,
        role TEXT NOT NULL,
        added_by_user_id INTEGER NULL REFERENCES users (id),
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        UNIQUE (user_id, team_id)
    );
CREATE TABLE sessions (
        id INTEGER PRIMARY KEY,
        user_id INTEGER NOT NULL REFERENCES users (id),
        token TEXT NOT NULL UNIQUE,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
CREATE TABLE connections (
        id INTEGER PRIMARY KEY,
        subdomain TEXT NOT NULL,
        team_member_id INTEGER NOT NULL REFERENCES team_members (id),
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        closed_at TIMESTAMP NULL
    );
CREATE TABLE global_settings (
        id INTEGER PRIMARY KEY,
        smtp_enabled BOOLEAN NOT NULL DEFAULT false,
        smtp_host TEXT NULL,
        smtp_port INTEGER NULL,
        smtp_username TEXT NULL,
        smtp_password TEXT NULL,
        from_address TEXT NULL,
        add_member_email_subject TEXT NULL,
        add_member_email_template TEXT NULL
    );
CREATE TABLE schema_migrations (version uint64,dirty bool);
CREATE UNIQUE INDEX version_unique ON schema_migrations (version);
CREATE TABLE migrations (
			id INT8 NOT NULL,
			version VARCHAR(255) NOT NULL,
			PRIMARY KEY (id)
		);
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  (20231230090812);
