DROP INDEX IF EXISTS idx_job_items_batch_id;
ALTER TABLE job_items DROP CONSTRAINT job_items_job_or_batch;
ALTER TABLE job_items DROP COLUMN batch_id;
ALTER TABLE job_items ALTER COLUMN job_id SET NOT NULL;
DROP TABLE batches;
