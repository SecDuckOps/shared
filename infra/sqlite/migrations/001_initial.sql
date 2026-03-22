-- sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT    PRIMARY KEY,
    user_id     TEXT    NOT NULL,
    device_id   TEXT    NOT NULL,
    name        TEXT    NOT NULL,
    status      TEXT    NOT NULL DEFAULT 'active',        
    metadata    TEXT    NOT NULL DEFAULT '{}',            -- JSON map[string]string
    created_at  INTEGER NOT NULL,                         -- Unix timestamp (UnixNano)
    updated_at  INTEGER NOT NULL,                         -- Unix timestamp (UnixNano)
    deleted_at  INTEGER,                                  -- nullable soft delete, UnixNano
    version     INTEGER NOT NULL DEFAULT 1,
    sync_status TEXT    NOT NULL DEFAULT 'local_only'     
);

-- messages table
CREATE TABLE IF NOT EXISTS messages (
    id          TEXT    PRIMARY KEY,
    session_id  TEXT    NOT NULL,
    role        TEXT    NOT NULL,                            
    content     TEXT    NOT NULL,
    created_at  INTEGER NOT NULL,                         -- Unix timestamp (UnixNano)
    sync_status TEXT    NOT NULL DEFAULT 'local_only',
    FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- checkpoints table
CREATE TABLE IF NOT EXISTS checkpoints (
    id            TEXT    PRIMARY KEY,
    session_id    TEXT    NOT NULL,
    message_index INTEGER NOT NULL,
    summary       TEXT    NOT NULL,
    created_at    INTEGER NOT NULL,                       -- Unix timestamp (UnixNano)
    sync_status   TEXT    NOT NULL DEFAULT 'local_only',
    FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- sync_queue table
CREATE TABLE IF NOT EXISTS sync_queue (
    id         TEXT    PRIMARY KEY,
    table_name TEXT    NOT NULL,
    record_id  TEXT    NOT NULL,
    operation  TEXT    NOT NULL,                          
    attempts   INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,                          -- Unix timestamp (UnixNano)
    last_error TEXT    NOT NULL DEFAULT ''
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_messages_session_id    ON messages(session_id);
CREATE INDEX IF NOT EXISTS idx_checkpoints_session_id ON checkpoints(session_id);
CREATE INDEX IF NOT EXISTS idx_sync_queue_operation   ON sync_queue(operation);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id       ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_deleted_at    ON sessions(deleted_at);
