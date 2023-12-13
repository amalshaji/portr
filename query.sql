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
    connections.*,
    users.*
FROM
    connections
    JOIN team_members ON team_members.id = connections.team_member_id
    JOIN users ON users.id = team_members.user_id
WHERE
    team_id = ?
    AND closed_at IS NULL
ORDER BY
    connections.id DESC
LIMIT
    20;

-- name: GetRecentConnectionsForTeam :many
SELECT
    connections.*,
    users.*
FROM
    connections
    JOIN team_members ON team_members.id = connections.team_member_id
    JOIN users ON users.id = team_members.user_id
WHERE
    team_id = ?
ORDER BY
    connections.id DESC
LIMIT
    20;

-- name: CreateNewConnection :one
INSERT INTO
    connections (subdomain, team_member_id)
VALUES
    (?, ?) RETURNING *;

-- name: MarkConnectionAsClosed :exec
UPDATE connections
SET
    closed_at = CURRENT_TIMESTAMP
WHERE
    id = ?;

-- name: GetInvitesForTeam :many
SELECT
    invites.email,
    invites.role,
    invites.status,
    users.email AS invited_by_email
FROM
    invites
    JOIN team_members ON team_members.id = invites.invited_by_team_member_id
    JOIN users ON users.id = team_members.user_id
WHERE
    invites.team_id = ?;

-- name: GetAtiveInviteByEmail :one
SELECT
    *
FROM
    invites
WHERE
    email = ?
    AND team_id = ?
    AND status = 'active';

-- name: CreateInvite :one
INSERT INTO
    invites (
        email,
        role,
        status,
        invited_by_team_member_id,
        team_id
    )
VALUES
    (?, ?, ?, ?, ?) RETURNING *;

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
        user_invite_email_subject,
        user_invite_email_template
    )
VALUES
    (?, ?) RETURNING *;

-- name: UpdateGlobalSettings :exec
UPDATE global_settings
SET
    user_invite_email_subject = ?,
    user_invite_email_template = ?;

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

-- name: GetActiveTeamInvitesForUser :many
SELECT
    *
FROM
    invites
WHERE
    email = ?
    AND status = 'active';

-- name: GetNumberOfExistingTeamInvitesForUser :one
SELECT
    COUNT(*)
FROM
    invites
WHERE
    email = ?
    AND team_id = ?
    AND status = 'active';

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE
    token = ?;

-- name: UpdateUser :exec
UPDATE users
SET
    first_name = ?,
    last_name = ?
WHERE
    id = ?;

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

-- name: AcceptInvite :exec
UPDATE invites
SET
    status = 'accepted'
WHERE
    id = ?;

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