CREATE TABLE shard_mappings (
  shard_key STRING(50) NOT NULL,
  shard_name STRING(50) NOT NULL,
) PRIMARY KEY (shard_key);
