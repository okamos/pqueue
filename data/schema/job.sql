CREATE TABLE IF NOT EXISTS "job" (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  payload bytea,
  status smallint, -- 0 yet, 1 done, 2 failed
  priority smallint,
  run_after timestamp with time zone,
  timeout smallint,
  run_count smallint,
  retry_delay smallint,
  grabbed timestamp with time zone,
  elapsed real,
  last_error text
);

CREATE INDEX IF NOT EXISTS "job_next_at_key" ON "job" (run_after);
