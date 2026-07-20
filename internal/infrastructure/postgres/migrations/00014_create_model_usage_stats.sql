-- +goose Up
CREATE TABLE model_usage_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model_id VARCHAR(255) NOT NULL,
    token_count INTEGER NOT NULL DEFAULT 0,
    first_token_time_ms INTEGER,
    inference_time_ms INTEGER NOT NULL DEFAULT 0,
    tokens_per_second NUMERIC(10,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_model_usage_stats_user_id ON model_usage_stats(user_id);
CREATE INDEX idx_model_usage_stats_user_created ON model_usage_stats(user_id, created_at);
CREATE INDEX idx_model_usage_stats_user_model ON model_usage_stats(user_id, model_id);
CREATE INDEX idx_model_usage_stats_model_id ON model_usage_stats(model_id);

-- +goose Down
DROP TABLE IF EXISTS model_usage_stats;
