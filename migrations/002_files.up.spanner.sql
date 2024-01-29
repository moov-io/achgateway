CREATE TABLE files (
  file_id STRING(MAX) NOT NULL,
  shard_key STRING(MAX) NOT NULL,
  hostname STRING(MAX) NOT NULL,
  accepted_at TIMESTAMP NOT NULL,
  canceled_at TIMESTAMP,
) PRIMARY KEY (file_id);
