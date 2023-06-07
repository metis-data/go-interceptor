-- Create schema
CREATE SCHEMA IF NOT EXISTS my_schema;

-- Create table
CREATE TABLE IF NOT EXISTS my_schema.my_table (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL
);

-- Insert sample data
INSERT INTO my_schema.my_table (name) VALUES
  ('Sample Data 1'),
  ('Sample Data 2');
