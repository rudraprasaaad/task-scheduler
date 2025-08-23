CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tasks table
CREATE TABLE
	tasks (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		type VARCHAR(100) NOT NULL,
		payload JSONB,
		priority INTEGER NOT NULL DEFAULT 5,
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		retries INTEGER NOT NULL DEFAULT 0,
		max_retries INTEGER NOT NULL DEFAULT 3,
		created_at TIMESTAMP
		WITH
			TIME ZONE NOT NULL DEFAULT NOW (),
			updated_at TIMESTAMP
		WITH
			TIME ZONE NOT NULL DEFAULT NOW (),
			scheduled_at TIMESTAMP
		WITH
			TIME ZONE NOT NULL DEFAULT NOW (),
			started_at TIMESTAMP
		WITH
			TIME ZONE,
			completed_at TIMESTAMP
		WITH
			TIME ZONE,
			error TEXT,
			worker_id VARCHAR(100)
	);

-- Workers table
CREATE TABLE
	workers (
		id VARCHAR(100) PRIMARY KEY,
		status VARCHAR(50) NOT NULL DEFAULT "idle",
		last_seen TIMESTAMP
		WITH
			TIME ZONE NOT NULL DEFAULT NOW (),
			tasks_run INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP
		WITH
			TIME ZONE NOT NULL DEFAULT NOW ()
	);

CREATE TABLE
	task_executions (
		id UUID DEFAULT uuid_generate_V4 () PRIMARY KEY,
		task_id VARCHAR(255) NOT NULL REFERENCES tasks (id),
		worker_id VARCHAR(100) REFERENCES workers (id),
		started_at TIMESTAMP
		WITH
			TIME ZONE NOT NULL,
			completed_at TIMESTAMP
		WITH
			TIME ZONE NOT NULL,
			status VARCHAR(50) NOT NULL,
			error TEXT,
			execution_time_ms INTEGER
	);

-- indexes
CREATE INDEX idx_tasks_status ON tasks (status);

CREATE INDEX idx_tasks_scheduled_at ON tasks (scheduled_at);

CREATE INDEX idx_tasks_priority ON tasks (priority);

CREATE INDEX idx_tasks_type ON tasks (type);

CREATE INDEX idx_workers_status ON workers (status);

CREATE INDEX idx_tasks_executions_task_id ON task_executions (task_id);

-- function to update updated_at column

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
	NEW.updated_at = NOW();
	RETURN NEW;
END;
$$ language 'plpgsql';

-- trigger to automatically update updated_at
CREATE TRIGGER update_tasks_updated_at BEFORE UPDATE ON tasks FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();