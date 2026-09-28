CREATE TABLE activity_logs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    event VARCHAR(64) NOT NULL,
    actor_id UUID,
    subject_type VARCHAR(32) NOT NULL,
    subject_id UUID NOT NULL,
    changes JSONB NOT NULL DEFAULT '{}',
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_activity_logs_created_at ON activity_logs (created_at DESC);
CREATE INDEX idx_activity_logs_subject ON activity_logs (subject_type, subject_id, created_at DESC);
CREATE INDEX idx_activity_logs_actor ON activity_logs (actor_id, created_at DESC);
CREATE INDEX idx_activity_logs_event ON activity_logs (event, created_at DESC);
