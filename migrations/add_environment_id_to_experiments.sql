-- Add environment_id column to experiments table
ALTER TABLE experiments 
ADD COLUMN environment_id uuid NOT NULL REFERENCES environments(id) ON DELETE CASCADE;

-- Create index for environment_id
CREATE INDEX idx_experiments_environment_id ON experiments(environment_id);

-- Note: This migration is handled automatically by GORM AutoMigrate
-- This file is for documentation purposes only
