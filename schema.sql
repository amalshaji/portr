CREATE TABLE
    IF NOT EXISTS users (
        id INTEGER PRIMARY KEY,
        email TEXT NOT NULL UNIQUE,
        first_name TEXT NULL,
        last_name TEXT NULL,
        is_super_user BOOLEAN NOT NULL DEFAULT false,
        github_access_token TEXT NULL,
        github_avatar_url TEXT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE
    IF NOT EXISTS teams (
        id INTEGER PRIMARY KEY,
        NAME TEXT NOT NULL UNIQUE,
        slug TEXT NOT NULL UNIQUE,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE
    IF NOT EXISTS team_members (
        id INTEGER PRIMARY KEY,
        user_id INTEGER NOT NULL REFERENCES users (id),
        team_id INTEGER NOT NULL REFERENCES teams (id),
        secret_key TEXT NOT NULL,
        role TEXT NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        UNIQUE (user_id, team_id)
    );

CREATE TABLE
    IF NOT EXISTS sessions (
        id INTEGER PRIMARY KEY,
        user_id INTEGER NOT NULL REFERENCES users (id),
        token TEXT NOT NULL UNIQUE,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE
    IF NOT EXISTS invites (
        id INTEGER PRIMARY KEY,
        email TEXT NOT NULL,
        role TEXT NOT NULL,
        status TEXT NOT NULL,
        invited_by_team_member_id INTEGER NOT NULL REFERENCES team_members (id),
        team_id INTEGER NOT NULL REFERENCES teams (id),
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        UNIQUE (email, team_id)
    );

CREATE TABLE
    IF NOT EXISTS connections (
        id INTEGER PRIMARY KEY,
        subdomain TEXT NOT NULL,
        team_member_id INTEGER NOT NULL REFERENCES team_members (id),
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        closed_at TIMESTAMP NULL
    );

CREATE TABLE
    IF NOT EXISTS global_settings (
        id INTEGER PRIMARY KEY,
        smtp_enabled BOOLEAN NOT NULL DEFAULT false,
        smtp_host TEXT NULL,
        smtp_port INTEGER NULL,
        smtp_username TEXT NULL,
        smtp_password TEXT NULL,
        from_address TEXT NULL,
        user_invite_email_subject TEXT NULL,
        user_invite_email_template TEXT NULL
    );