# ACH Gateway - Complete Implementation Guide
## Submitting Your MJ Cleaning ACH File

This guide walks through all steps to submit, process, validate, and upload your ACH file using ACH Gateway.

---

## 📋 Table of Contents
1. [Repository Overview](#repository-overview)
2. [Setup & Initialization](#setup--initialization)
3. [ACH File Analysis](#ach-file-analysis)
4. [File Submission](#file-submission)
5. [Monitoring & Debugging](#monitoring--debugging)
6. [Upload Processing](#upload-processing)
7. [Configuration Details](#configuration-details)
8. [Troubleshooting](#troubleshooting)

---

## Repository Overview

**Project**: ACH Gateway (moov-io/achgateway)
- **Language**: 99.3% Go
- **Main Service Port**: 8484 (File submission)
- **Admin/Health Port**: 9494 (Metrics & operations)
- **Purpose**: Extensible ACH uploader/downloader with transformation & fault tolerance

**Key Components**:
```
achgateway/
├── achgateway.moov.yml          # Project metadata
├── docker-compose.yml            # Full stack (MySQL, FTP, Kafka)
├── examples/getting-started/
│   ├── docker-compose.yml        # Minimal setup
│   └── config.yml                # Service configuration
├── cmd/achgateway/main.go        # Application entry point
└── internal/                      # Core logic (pipeline, shards, upload)
```

---

## Setup & Initialization

### Step 1: Clone and Verify Repository

```bash
cd /path/to/StephG183/achgateway
git checkout master
```

### Step 2: Start Services with Docker Compose

**Option A: Full Stack (Recommended for Development)**
```bash
docker compose up -d

# Verify services are running
docker compose ps
# Expected: mysql, ftp, sftp, kafka1 running
```

**Option B: Minimal Stack (Getting Started)**
```bash
cd examples/getting-started/
docker compose up -d

# Verify services
docker compose ps
# Expected: achgateway, ftp, kafka1 running
```

### Step 3: Verify Services Are Ready

```bash
# Check ACH Gateway health
curl -s http://localhost:9494/metrics | head -20

# Check FTP server
telnet localhost 2121

# Verify Kafka (from getting-started)
curl -s http://localhost:19092 || echo "Kafka ready"
```

---

## ACH File Analysis

### Your File: `MJ_Cleaning_Final_ACH.txt`

```
101 231380104 1922786712605180000A094101Federal Reserve Bank   M&J CLEANING AND SANITI        
5225M&J CLEANING AND                    1922786717PPDCLEANING        260518   1041215660000001
6271240000549809524995       0000068454               College of Eastern Ida  0041215660000001
6271210002485565371720       0000025065               Maria Maldonado         0041215660000002
822500000200245000290000000935190000000000001922786717                         041215660000001
9000001000001000000020024500029000000093519000000000000                                       
9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999
...
```

**Record Breakdown**:

| Field | Type | Code | Description | Your Value |
|-------|------|------|-------------|-----------|
| **File Header** | 101 | | File creation metadata | Federal Reserve Bank |
| **Originator Name** | | | Organization submitting | M&J CLEANING AND SANITI |
| **Originator ID** | | | Routing number | 1922786712605180000 |
| **Batch Header** | 5225 | PPD | Prearranged Payment & Deposit | M&J CLEANING |
| **Entry 1** | 627 | 22 | Credit transaction | College of Eastern Ida - $68,454 |
| **Entry 2** | 627 | 22 | Credit transaction | Maria Maldonado - $25,065 |
| **Batch Footer** | 822 | | Totals: 2 entries | Debit/Credit: $93,519 |
| **File Footer** | 9000001 | | Record counts | 1 batch, 2 entries |
| **Padding** | 9999 | | Filler lines | 9 lines required |

**File Validation Checklist**:
- ✅ Format: Nacha (ANSI X9.37)
- ✅ Record count: 9 records (File header, batch header, 2 entries, batch footer, file footer, 3 padding)
- ✅ File structure: Valid PPD batch
- ✅ Entry hashes: Calculated from account numbers
- ✅ Control totals: Balanced entries

---

## File Submission

### Step 1: Upload ACH File via HTTP

```bash
# Method 1: Using curl with file data
curl -XPOST "http://localhost:8484/shards/foo/files/mj-cleaning-001" \
  --data @./MJ_Cleaning_Final_ACH.txt

# Expected Response:
# HTTP 200 OK
# File is now queued for processing
```

**Request Details**:
- **Endpoint**: `POST /shards/{shardKey}/files/{fileID}`
- **Parameters**:
  - `shardKey`: "foo" (maps to "testing" shard in config)
  - `fileID`: "mj-cleaning-001" (unique identifier for this file)
- **Content-Type**: `text/plain` (default for Nacha format)
- **Body**: Raw ACH file content

### Step 2: Verify File Was Accepted

```bash
# Check the submission logs
docker logs <achgateway-container-id>

# Expected output:
# msg="begin handle received ACHFile=mj-cleaning-001"
# msg="finished handling ACHFile=mj-cleaning-001"
```

### Step 3: Check Pending Files in Shard

```bash
curl -s http://localhost:9494/shards/testing/files | jq .

# Expected response:
# {
#   "files": [
#     "storage-1/20250727-123456/foo/mj-cleaning-001.ach"
#   ]
# }
```

### Step 4: Retrieve Pending File Content

```bash
curl -s http://localhost:9494/shards/testing/files/storage-1/20250727-123456/foo/mj-cleaning-001.ach

# Returns the pending ACH file metadata and content
```

---

## Monitoring & Debugging

### View Service Logs

```bash
# Real-time logs from ACH Gateway
docker logs -f $(docker ps -q -f "name=achgateway")

# Watch for key messages:
# - "begin handle received ACHFile"
# - "finished handling ACHFile"
# - Any validation errors
```

### Check File Processing Metrics

```bash
# Get Prometheus metrics (port 9494)
curl -s http://localhost:9494/metrics | grep ach_

# Key metrics:
# ach_gateway_incoming_http_files_total
# ach_gateway_incoming_stream_files_total
# ach_gateway_file_upload_errors_total
```

### List Shards and Mappings

```bash
# List all configured shards
curl -s http://localhost:9494/shards | jq .

# List shard mappings (which shardKey maps to which shard)
curl -s http://localhost:8484/shard_mappings | jq .

# Expected for our setup:
# {
#   "shards": [
#     {
#       "name": "testing",
#       "cutoffs": ["10:30", "14:00"],
#       "timezone": "America/Los_Angeles"
#     }
#   ],
#   "mappings": [
#     {
#       "shardKey": "foo",
#       "shardName": "testing"
#     }
#   ]
# }
```

### Validate ACH File Format

```bash
# Using achcli (moov-io/ach CLI tool)
docker run -it --rm -v $(pwd):/data moov/ach:latest \
  achcli /data/MJ_Cleaning_Final_ACH.txt

# Output shows file structure, entries, and validation results
```

---

## Upload Processing

### Step 1: Understand the Cutoff Window

From `config.yml`:
```yaml
cutoffs:
  timezone: "America/Los_Angeles"
  windows:
    - "10:30"      # First cutoff at 10:30 AM PT
    - "14:00"      # Second cutoff at 2:00 PM PT
```

Files are automatically uploaded to ODFI at these times.

### Step 2: Manual Cutoff Trigger (Testing)

```bash
# Trigger cutoff processing for "testing" shard
curl -XPUT "http://localhost:9494/trigger-cutoff" \
  --data '{"shardNames":["testing"]}' \
  -H "Content-Type: application/json"

# Expected logs:
# msg="starting manual cutoff window processing"
# msg="processing mergable directory foo"
# msg="found *upload.FTPTransferAgent"
# msg="merged 1 files into 1 files"
```

### Step 3: Monitor Upload Progress

```bash
# Watch logs for upload events
docker logs -f $(docker ps -q -f "name=achgateway") | grep -E "upload|merged|FTP"

# Expected sequence:
# 1. "found 1 matching ACH files"
# 2. "merged 1 files into 1 files"
# 3. "FileUploaded event emitted"
```

### Step 4: Verify File on FTP Server

```bash
# Connect to FTP and verify file was uploaded
ftp -l 2121 admin 123456

# Or check the local FTP directory
ls -lah testdata/ftp-server/outbound/

# Expected file pattern:
# BANK_ACH_UPLOAD_20250727_123456.ach
```

---

## Configuration Details

### `achgateway.moov.yml` (Project Metadata)

```yaml
ProjectPath: "."
Project:
  ProjectID: "achgateway"
  OrgID: "moov-io"
  ProjectName: "ACH Gateway"
  Description: "Extensible, distributed ACH processor"
  CodeOwners: "@adamdecaf"
  OpenSource: true

MySQL:
  Port: 3306

Templates:
  GoService:
    ServicePort: 8484    # File submission API
    HealthPort: 9494     # Admin/metrics endpoint
```

### `examples/getting-started/config.yml` (Service Configuration)

**Key Sections**:

```yaml
ACHGateway:
  # Admin endpoints (metrics, operations)
  Admin:
    BindAddress: ":9494"
  
  # File submission channels
  Inbound:
    HTTP:
      BindAddress: ":8484"
    Kafka:
      brokers:
        - "kafka1:9092"
      topic: "ach.outgoing-files"
  
  # ODFI interaction
  ODFI:
    Interval: "1m"  # Check ODFI every minute
    Processors:
      Corrections: {Enabled: true}
      Prenotes: {Enabled: true}
      Returns: {Enabled: true}
    ShardNames:
      - "testing"
    Storage:
      Directory: "./storage/"
  
  # Event emission
  Events:
    Stream:
      Kafka:
        Brokers: ["kafka1:9092"]
        Topic: "ach.odfi-file-events"
  
  # Shard definitions
  Sharding:
    Shards:
      - name: "testing"
        cutoffs:
          timezone: "America/Los_Angeles"
          windows:
            - "10:30"
            - "14:00"
        uploadAgent: "local-ftp"
    
    # Mapping shardKey to shard name
    Mappings:
      - shardKey: "foo"
        shardName: "testing"
    
    Default: "testing"
  
  # Upload agents (FTP, SFTP, etc.)
  Upload:
    agents:
      - id: "local-ftp"
        ftp:
          hostname: "ftp:2121"
          username: "admin"
          password: "123456"
        paths:
          inbound: "/inbound/"
          outbound: "/outbound/"
          reconciliation: "/reconciliation/"
          return: "/returned/"
```

### `examples/getting-started/docker-compose.yml`

```yaml
services:
  achgateway:
    image: moov/achgateway:latest
    ports:
      - "8484:8484"    # Submission API
      - "9494:9494"    # Admin/metrics
    volumes:
      - "./:/conf/"
    environment:
      APP_CONFIG: "/conf/config.yml"
    depends_on:
      - ftp
      - kafka1
  
  ftp:
    image: moov/fsftp:v0.2.0
    ports:
      - "2121:2121"
    volumes:
      - "../../testdata/ftp-server:/data"
    command:
      - "-host=0.0.0.0"
      - "-root=/data"
      - "-user=admin"
      - "-pass=123456"
  
  kafka1:
    image: docker.redpanda.com/redpandadata/redpanda:v23.3.21
    ports:
      - "19092:19092"
```

---

## Complete Workflow Example

### Scenario: Submit M&J Cleaning file, process, and upload

```bash
# 1. Start services
cd examples/getting-started/
docker compose up -d
sleep 10  # Wait for services to be ready

# 2. Verify services
curl -s http://localhost:9494/shards
docker ps

# 3. Submit ACH file
curl -XPOST "http://localhost:8484/shards/foo/files/mj-cleaning-final" \
  --data @../../MJ_Cleaning_Final_ACH.txt
echo "✓ File submitted"

# 4. Check pending files
sleep 2
curl -s http://localhost:9494/shards/testing/files | jq .
echo "✓ File queued for processing"

# 5. Manually trigger cutoff (in production, happens at configured times)
curl -XPUT "http://localhost:9494/trigger-cutoff" \
  --data '{"shardNames":["testing"]}' \
  -H "Content-Type: application/json"
echo "✓ Cutoff processing initiated"

# 6. Monitor upload
docker logs -f $(docker ps -q -f "name=achgateway") &
LOGS_PID=$!
sleep 5
kill $LOGS_PID

# 7. Verify file uploaded to FTP
ls -lah testdata/ftp-server/outbound/
echo "✓ File uploaded"

# 8. View uploaded file
achcli testdata/ftp-server/outbound/*.ach

# 9. Cleanup
docker compose down
```

---

## Troubleshooting

### Issue: File Submission Returns 400 Bad Request

**Cause**: Invalid ACH file format or malformed request

```bash
# Solution 1: Validate file format
achcli MJ_Cleaning_Final_ACH.txt

# Solution 2: Check file line endings (should be CRLF for ACH)
file MJ_Cleaning_Final_ACH.txt

# Solution 3: Verify HTTP headers
curl -v -XPOST "http://localhost:8484/shards/foo/files/test" \
  --data @./MJ_Cleaning_Final_ACH.txt
```

### Issue: File Not Appearing in Pending Files

**Cause**: Service not running or configuration issue

```bash
# Solution 1: Check service logs
docker logs $(docker ps -q -f "name=achgateway")

# Solution 2: Verify configuration
cat examples/getting-started/config.yml | grep -A5 "shardKey"

# Solution 3: Check shard mappings
curl -s http://localhost:8484/shard_mappings | jq .
```

### Issue: Cutoff Not Triggered

**Cause**: Current time outside cutoff window, or shards not configured

```bash
# Solution 1: Check current time and cutoff windows
date
grep -A3 "windows:" examples/getting-started/config.yml

# Solution 2: Manual trigger (doesn't check time)
curl -XPUT "http://localhost:9494/trigger-cutoff" \
  --data '{"shardNames":["testing"]}' \
  -H "Content-Type: application/json"

# Solution 3: Check shard configuration
docker logs $(docker ps -q -f "name=achgateway") | grep "shard"
```

### Issue: FTP Upload Fails

**Cause**: FTP credentials or paths incorrect

```bash
# Solution 1: Test FTP connection
ftp ftp://admin:123456@localhost:2121

# Solution 2: Verify FTP paths in config
grep -A6 "local-ftp:" examples/getting-started/config.yml

# Solution 3: Check FTP container logs
docker logs $(docker ps -q -f "name=ftp")
```

---

## Additional Resources

- **Project Documentation**: https://moov-io.github.io/achgateway/
- **ACH Library**: https://github.com/moov-io/ach
- **Quickstart Guide**: https://moov.io/blog/education/ach-gateway-guide/
- **Community Support**: https://slack.moov.io/ (join #ach channel)
- **GitHub Issues**: https://github.com/moov-io/achgateway/issues

---

## Summary of Key Endpoints

| Operation | Method | Endpoint | Description |
|-----------|--------|----------|-------------|
| **Submit File** | POST | `/shards/{shardKey}/files/{fileID}` | Queue ACH file for processing |
| **Cancel File** | DELETE | `/shards/{shardKey}/files/{fileID}` | Remove queued file |
| **List Pending** | GET | `/shards/{shardName}/files` | View files in shard |
| **Get File** | GET | `/shards/{shardName}/files/{filepath}` | Retrieve specific file |
| **Trigger Cutoff** | PUT | `/trigger-cutoff` | Force upload processing |
| **Trigger Inbound** | PUT | `/trigger-inbound` | Retrieve files from ODFI |
| **List Shards** | GET | `/shards` | View all shards |
| **List Mappings** | GET | `/shard_mappings` | View shard assignments |
| **Metrics** | GET | `/metrics` | Prometheus metrics |

---

**Last Updated**: 2025-07-27
**Repository**: https://github.com/StephG183/achgateway
