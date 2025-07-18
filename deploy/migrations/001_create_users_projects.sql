-- users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username STRING UNIQUE NOT NULL,
    hashed_password STRING NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- projects table
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name STRING NOT NULL,
    searchable_keys STRING[] NOT NULL,
    api_key STRING UNIQUE NOT NULL,
    ttl_days INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- membership (which users can see which projects)
CREATE TABLE IF NOT EXISTS project_members (
    user_id UUID NOT NULL REFERENCES users(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    PRIMARY KEY (user_id, project_id)
);