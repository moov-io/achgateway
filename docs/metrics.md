---
layout: page
title: Prometheus Metrics
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Prometheus Metrics

ACHGateway emits Prometheus metrics on the admin HTTP server at `/metrics`. Typically [Alertmanager](https://github.com/prometheus/alertmanager) is set up to aggregate the metrics and alert teams.

### HTTP Server

- `http_response_duration_seconds`: Histogram representing the http response durations

### Database

- `mysql_connections`: How many MySQL connections and what status they're in.
- `sqlite_connections`: How many sqlite connections and what status they're in.

### ODFI Files

- `correction_codes_processed`: Counter of correction (COR/NOC) files processed
- `files_downloaded`: Counter of files downloaded from a remote server
- `missing_return_transfers`: Counter of return EntryDetail records handled without a fund transfer
- `prenote_entries_processed`: Counter of prenote EntryDetail records processed
- `return_entries_processed`: Counter of return EntryDetail records processed


## Incoming Files

- `incoming_http_files`: Counter of ACH files submitted through the http interface
- `incoming_stream_files`: Counter of ACH files received through stream interface
- `http_file_processing_errors`: Counter of http submitted ACH files that failed processing
- `stream_file_processing_errors`: Counter of stream submitted ACH files that failed processing

## Outbound Files

- `pending_files`: Counter of ACH files waiting to be uploaded
- `files_missing_shard_aggregators`: Counter of ACH files unable to be matched with a shard aggregator
- `ach_uploaded_files`: Counter of ACH files uploaded through the pipeline to the ODFI
- `ach_upload_errors`: Counter of errors encountered when attempting ACH files upload

### Remote File Servers

- `ftp_agent_up`: Status of FTP agent connection
- `sftp_agent_up`: Status of SFTP agent connection
