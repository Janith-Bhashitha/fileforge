CREATE TABLE batches (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation TEXT NOT NULL,
    total INT NOT NULL DEFAULT 0,
    completed INT NOT NULL DEFAULT 0,
    failed INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'processing',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_batches_owner_id ON batches (owner_id);

-- A job_item now belongs to exactly one of: a single Job (Phase 3) or a
-- Batch (Phase 4) - never both, never neither.
ALTER TABLE job_items ALTER COLUMN job_id DROP NOT NULL;
ALTER TABLE job_items ADD COLUMN batch_id UUID REFERENCES batches(id) ON DELETE CASCADE;
ALTER TABLE job_items ADD CONSTRAINT job_items_job_or_batch
    CHECK ((job_id IS NOT NULL) != (batch_id IS NOT NULL));

CREATE INDEX idx_job_items_batch_id ON job_items (batch_id);
