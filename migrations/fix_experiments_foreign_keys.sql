-- Remove wrong FK that points to flag_rules
ALTER TABLE experiments DROP CONSTRAINT IF EXISTS fk_flag_rules_experiment;
ALTER TABLE experiments DROP CONSTRAINT IF EXISTS fk_experiments_flag_rules_experiment;

-- Ensure correct FK to flags table
ALTER TABLE experiments DROP CONSTRAINT IF EXISTS fk_experiments_flag;
ALTER TABLE experiments ADD CONSTRAINT fk_experiments_flag FOREIGN KEY (flag_id) REFERENCES flags(id) ON DELETE CASCADE;

-- Ensure correct FK to environments table
ALTER TABLE experiments DROP CONSTRAINT IF EXISTS fk_experiments_environment;
ALTER TABLE experiments ADD CONSTRAINT fk_experiments_environment FOREIGN KEY (environment_id) REFERENCES environments(id) ON DELETE CASCADE;
