-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id              BIGSERIAL PRIMARY KEY,
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    username        TEXT NOT NULL,
    bio             TEXT NOT NULL DEFAULT '',
    avatar_url      TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_admin        BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS skills (
    id   BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS tags (
    id        BIGSERIAL PRIMARY KEY,
    name      TEXT NOT NULL UNIQUE,
    is_system BOOL NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS user_skills (
    user_id  BIGINT NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    skill_id BIGINT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, skill_id)
);

CREATE TABLE IF NOT EXISTS projects (
    id              BIGSERIAL PRIMARY KEY,
    owner_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'recruiting'
                        CHECK (status IN ('draft', 'recruiting', 'in_progress', 'completed', 'archived', 'banned')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ban_reason      TEXT NOT NULL DEFAULT '',
    appeal_status   TEXT NOT NULL DEFAULT 'none',
                        CHECK (appeal_status IN ('none','pending','approved','rejected')),
    appeal_comment  TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS project_tags (
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tag_id     BIGINT NOT NULL REFERENCES tags(id)     ON DELETE CASCADE,
    PRIMARY KEY (project_id, tag_id)
);

CREATE TABLE IF NOT EXISTS project_roles (
    id          BIGSERIAL PRIMARY KEY,
    project_id  BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    role_name   TEXT NOT NULL DEFAULT '',
    grade       TEXT NOT NULL DEFAULT '',
    slots_total INT  NOT NULL CHECK (slots_total > 0),
    slots_filled INT NOT NULL DEFAULT 0 CHECK (slots_filled >= 0 AND slots_filled <= slots_total),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_project_roles_name_grade UNIQUE (project_id, role_name, grade)
);

CREATE TABLE IF NOT EXISTS applications (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id)         ON DELETE CASCADE,
    project_role_id BIGINT NOT NULL REFERENCES project_roles(id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'accepted', 'rejected', 'withdrawn')),
    message         TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS project_members (
    id              BIGSERIAL PRIMARY KEY,
    project_id      BIGINT NOT NULL REFERENCES projects(id)      ON DELETE CASCADE,
    user_id         BIGINT NOT NULL REFERENCES users(id)         ON DELETE CASCADE,
    project_role_id BIGINT NOT NULL REFERENCES project_roles(id) ON DELETE CASCADE,
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_project_members_user_role UNIQUE (project_id, user_id, project_role_id)
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    jti        TEXT NOT NULL UNIQUE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_users_email            ON users (email);
CREATE UNIQUE INDEX IF NOT EXISTS ux_applications_active    ON applications (user_id, project_role_id) WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_projects_owner_id            ON projects(owner_id);
CREATE INDEX IF NOT EXISTS idx_projects_status              ON projects(status);
CREATE INDEX IF NOT EXISTS idx_project_tags_tag_id          ON project_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_project_roles_project_id     ON project_roles(project_id);
CREATE INDEX IF NOT EXISTS idx_applications_project_role_id ON applications(project_role_id);
CREATE INDEX IF NOT EXISTS idx_applications_user_id         ON applications(user_id);
CREATE INDEX IF NOT EXISTS idx_project_members_project_id   ON project_members(project_id);
CREATE INDEX IF NOT EXISTS idx_project_members_user_id      ON project_members(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id       ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at    ON refresh_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_revoked_at    ON refresh_tokens(revoked_at);

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS applications;
DROP TABLE IF EXISTS project_roles;
DROP TABLE IF EXISTS project_tags;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS user_skills;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS skills;
DROP TABLE IF EXISTS users;
