-- ---------------------------------------------------------------------------
-- users: people who use the bot.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id           BIGSERIAL PRIMARY KEY,    -- auto-incrementing internal id
    github_id    BIGINT NOT NULL UNIQUE,   -- GitHub's stable user id (from OAuth) — the identity
    email        TEXT UNIQUE NOT NULL,     -- where we send notifications
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- repos: every repository we poll
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS repos (
    id             BIGSERIAL PRIMARY KEY,
    owner          TEXT NOT NULL,
    name           TEXT NOT NULL,

    -- Conditional-request cache: GitHub returns an ETag; sending it back as
    -- If-None-Match lets a "nothing changed" reply be free (HTTP 304, no rate cost).
    etag           TEXT,

    last_polled_at TIMESTAMPTZ,
    -- Adaptive polling: active repos get a short interval, quiet ones back off.
    poll_interval  INTERVAL NOT NULL DEFAULT INTERVAL '15 minutes',
    -- When this repo is next due for a poll. The scheduler picks the earliest.
    next_poll_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (owner, name)
);

-- The scheduler repeatedly asks "which repos are due to poll?" — this index
-- makes that ORDER BY next_poll_at lookup fast even with many thousands of repos.
CREATE INDEX IF NOT EXISTS idx_repos_next_poll ON repos (next_poll_at);

-- ---------------------------------------------------------------------------
-- subscriptions: which user follows which repo (many-to-many).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS subscriptions (
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    repo_id     BIGINT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, repo_id)
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_repo ON subscriptions (repo_id);

CREATE TABLE IF NOT EXISTS notified (
    user_id          BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    repo_id          BIGINT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    github_issue_id  BIGINT NOT NULL,        -- GitHub's global issue id
    notified_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, github_issue_id)
);
