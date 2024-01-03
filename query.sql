-- name: GetUserBySession :one
SELECT
    users.id,
    users.email,
    users.created_at,
    users.first_name,
    users.last_name,
    users.github_avatar_url,
    users.is_super_user
FROM
    users
    JOIN sessions ON sessions.user_id = users.id
WHERE
    sessions.token = ?
LIMIT
    1;

-- name: GetTeamMemberByEmail :one
SELECT
    *
FROM
    team_members
    JOIN users ON users.id = team_members.user_id
WHERE
    users.email = ?
LIMIT
    1;

-- name: GetTeamMemberByUserIdAndTeamSlug :one
SELECT
    team_members.*,
    users.*
FROM
    team_members
    JOIN users ON users.id = team_members.user_id
    JOIN teams ON teams.id = team_members.team_id
WHERE
    users.id = ?
    AND teams.slug = ?
LIMIT
    1;

-- name: CreateTeam :one
INSERT INTO
    teams (name, slug)
VALUES
    (?, ?) RETURNING *;

-- name: CreateTeamMember :one
INSERT INTO
    team_members (user_id, team_id, role, secret_key)
VALUES
    (?, ?, ?, ?) RETURNING *;

-- name: CreateSession :one
INSERT INTO
    sessions (token, user_id)
VALUES
    (?, ?) RETURNING *;

-- name: GetUsersCount :one
SELECT
    COUNT(*)
FROM
    users;

-- name: GetTeamUserBySecretKey :one
SELECT
    *
FROM
    team_members
WHERE
    secret_key = ?
LIMIT
    1;

-- name: GetActiveConnectionsForTeam :many
SELECT
    connections.id,
    connections.type,
    connections.port,
    connections.subdomain,
    connections.created_at,
    connections.started_at,
    connections.closed_at,
    connections.status,
    users.email,
    users.first_name,
    users.last_name,
    users.github_avatar_url
FROM
    connections
    JOIN team_members ON team_members.id = connections.team_member_id
    JOIN users ON users.id = team_members.user_id
WHERE
    connections.team_id = ?
    AND status = 'active'
ORDER BY
    connections.id DESC
LIMIT
    20;

-- name: GetRecentConnectionsForTeam :many
SELECT
    connections.id,
    connections.type,
    connections.port,
    connections.subdomain,
    connections.created_at,
    connections.started_at,
    connections.closed_at,
    connections.status,
    users.email,
    users.first_name,
    users.last_name,
    users.github_avatar_url
FROM
    connections
    JOIN team_members ON team_members.id = connections.team_member_id
    JOIN users ON users.id = team_members.user_id
WHERE
    connections.team_id = ?
    AND status != 'reserved'
ORDER BY
    connections.id DESC
LIMIT
    20;

-- name: CreateNewHttpConnection :one
INSERT INTO
    connections (id, type, subdomain, team_member_id, team_id)
VALUES
    (?, "http", ?, ?, ?) RETURNING *;

-- name: CreateNewTcpConnection :one
INSERT INTO
    connections (id, type, port, team_member_id, team_id)
VALUES
    (?, "tcp", ?, ?, ?) RETURNING *;

-- name: MarkConnectionAsActive :exec
UPDATE connections
SET
    status = 'active',
    started_at = CURRENT_TIMESTAMP
WHERE
    id = ?;

-- name: MarkConnectionAsClosed :exec
UPDATE connections
SET
    status = 'closed',
    closed_at = CURRENT_TIMESTAMP
WHERE
    id = ?;

-- name: GetGlobalSettings :one
SELECT
    *
FROM
    global_settings
LIMIT
    1;

-- name: CreateGlobalSettings :one
INSERT INTO
    global_settings (
        smtp_enabled,
        add_member_email_subject,
        add_member_email_template
    )
VALUES
    (?, ?, ?) RETURNING *;

-- name: UpdateGlobalSettings :exec
UPDATE global_settings
SET
    smtp_enabled = ?,
    smtp_host = ?,
    smtp_port = ?,
    smtp_username = ?,
    smtp_password = ?,
    from_address = ?,
    add_member_email_subject = ?,
    add_member_email_template = ?;

-- name: GetTeamMembers :many
SELECT
    users.email,
    team_members.role,
    users.github_avatar_url
FROM
    team_members
    JOIN users ON users.id = team_members.user_id
WHERE
    team_id = ?;

-- name: CreateUser :one
INSERT INTO
    users (
        email,
        first_name,
        last_name,
        is_super_user,
        github_access_token,
        github_avatar_url
    )
VALUES
    (?, ?, ?, ?, ?, ?) RETURNING *;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE
    token = ?;

-- name: UpdateSecretKey :exec
UPDATE team_members
SET
    secret_key = ?
WHERE
    id = ?;

-- name: GetUserByEmail :one
SELECT
    *
FROM
    users
WHERE
    email = ?
LIMIT
    1;

-- name: GetTeamsOfUser :many
SELECT
    teams.*
FROM
    team_members
    JOIN teams ON teams.id = team_members.team_id
WHERE
    team_members.user_id = ?;

-- name: GetTeamMemberById :one
SELECT
    team_members.*,
    users.*
FROM
    team_members
    JOIN users ON users.id = team_members.user_id
WHERE
    team_members.id = ?
LIMIT
    1;

-- name: GetUserById :one
SELECT
    users.id,
    users.email,
    users.created_at,
    users.first_name,
    users.last_name,
    users.github_avatar_url,
    users.is_super_user
FROM
    users
WHERE
    id = ?
LIMIT
    1;

-- name: GetTeamById :one
SELECT
    *
FROM
    teams
WHERE
    id = ?
LIMIT
    1;

-- name: UpdateUser :exec
UPDATE users
SET
    first_name = COALESCE(?, first_name),
    last_name = COALESCE(?, last_name),
    github_access_token = COALESCE(?, github_access_token),
    github_avatar_url = COALESCE(?, github_avatar_url)
WHERE
    id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE
    strftime ('%s', 'now') - strftime ('%s', created_at) > 24 * 60 * 60;

-- name: DeleteUnclaimedConnections :exec
DELETE FROM connections
WHERE
    status = 'reserved'
    AND strftime ('%s', 'now') - strftime ('%s', created_at) > 10;

-- name: GetReservedOrActiveConnectionById :one
SELECT
    *
FROM
    connections
    JOIN team_members ON team_members.id = connections.team_member_id
WHERE
    connections.id = ?
    AND status IN ('active', 'reserved')
LIMIT
    1;

-- name: AddPortToConnection :exec
UPDATE connections
SET
    port = ?
WHERE
    id = ?;

-- name: GetReservedOrActiveConnectionForSubdomain :one
SELECT
    *
FROM
    connections
    JOIN team_members ON team_members.id = connections.team_member_id
WHERE
    subdomain = ?
    AND team_members.secret_key = ?
    AND status IN ('active', 'reserved')
LIMIT
    1;

-- name: GetReservedOrActiveConnectionForPort :one
SELECT
    *
FROM
    connections
    JOIN team_members ON team_members.id = connections.team_member_id
WHERE
    port = ?
    AND team_members.secret_key = ?
    AND status IN ('active', 'reserved')
LIMIT
    1;