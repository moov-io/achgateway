CREATE TABLE files(
       file_id VARCHAR(128) PRIMARY KEY,
       shard_key VARCHAR(50) NOT NULL,
       hostname VARCHAR(100) NOT NULL,
       accepted_at TIMESTAMP NOT NULL,
       canceled_at TIMESTAMP
);
