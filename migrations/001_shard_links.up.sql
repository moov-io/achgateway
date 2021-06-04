CREATE TABLE shard_mappings(
       shard_key VARCHAR(50) PRIMARY KEY,
       shard_name VARCHAR(50) NOT NULL,

       CONSTRAINT shard_mappings_unq_idx UNIQUE (shard_key, shard_name)
);
