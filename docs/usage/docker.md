---
layout: page
title: Usage | Docker
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Docker

You can download a [docker image called `moov/achgateway`](https://hub.docker.com/r/moov/achgateway/) from Docker Hub or use this repository. However it's recommended to [download the code repository](https://github.com/moov-io/achgateway) and running `docker-compose up` in the root directory.

```
$ docker-compose up achgateway
...
achgateway_1  | ts=2021-06-18T23:38:06Z msg="public listening on :8484" version=v0.4.1 level=info app=achgateway
achgateway_1  | ts=2021-06-18T23:38:06Z msg="listening on [::]:9494" version=v0.4.1 level=info app=achgateway
```

### Submitting a file

Submit a file to achgateway (Nacha ACH format):
```
$ curl -XPOST "http://localhost:8484/shards/foo/files/f6" --data @./testdata/ppd-debit.ach
...
achgateway_1  | ts=2021-06-18T23:38:16Z msg="begin handle received ACHFile=f6 of 1918 bytes" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:38:16Z msg="finished handling ACHFile=f6" level=info app=achgateway version=v0.4.1
```

Submit a file to achgateway (moov-io ACH JSON format):
```
$ curl -XPOST "http://localhost:8484/shards/foo/files/f4" --data @./testdata/ppd-valid.json
...
achgateway_1  | ts=2021-06-18T23:38:16Z msg="begin handle received ACHFile=f4 of 1918 bytes" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:38:16Z msg="finished handling ACHFile=f4" level=info app=achgateway version=v0.4.1
```

### Uploading files (to your ODFI)

Initiate cutoff time processing (aka upload to your ODFI):
```
$ curl -XPUT "http://localhost:9494/trigger-cutoff" --data '{"shardNames":["foo"]}'
achgateway_1  | ts=2021-06-18T23:38:20Z msg="starting manual cutoff window processing" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:38:20Z msg="processing mergable directory foo" level=info app=achgateway version=v0.4.1 shardKey=foo
achgateway_1  | ts=2021-06-18T23:38:20Z msg="found *upload.FTPTransferAgent agent" version=v0.4.1 shardKey=foo level=info app=achgateway
achgateway_1  | ts=2021-06-18T23:38:20Z msg="found 1 matching ACH files: []string{\"storage-1/20210618-233820/foo/f4.ach\"}" tenantID=foo level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:38:20Z msg="merged 1 files into 1 files" level=info app=achgateway version=v0.4.1 tenantID=foo
```

### View the uploaded file:

Using the [achcli](https://github.com/moov-io/ach#command-line) tool to view ACH files in a human readable format.

```
$ achcli ./testdata/ftp-server/outbound/20210618-233820-231380104.ach
Describing ACH file './testdata/ftp-server/outbound/20210618-233820-231380104.ach'

  Origin     OriginName   Destination  DestinationName  FileCreationDate  FileCreationTime
  121042882  Wells Fargo  231380104    Citadel          181008            0101

  BatchNumber  SECCode  ServiceClassCode                CompanyName  DiscretionaryData  Identification  EntryDescription  DescriptiveDate
  1            PPD      200 (Mixed Debits and Credits)  Wells Fargo                     121042882       Trans. Des

    TransactionCode       RDFIIdentification  AccountNumber      Amount  Name                    TraceNumber      Category
    22 (Checking Credit)  23138010            81967038518        100000  Steven Tander           121042880000001

    TransactionCode      RDFIIdentification  AccountNumber      Amount  Name                    TraceNumber      Category
    27 (Checking Debit)  12104288            17124411           100000  My ODFI                 121042880000002

  ServiceClassCode                EntryAddendaCount  EntryHash  TotalDebits  TotalCredits  MACCode  ODFIIdentification  BatchNumber
  200 (Mixed Debits and Credits)  2                  35242298   100000       100000                 12104288            1

  BatchCount  BlockCount  EntryAddendaCount  TotalDebitAmount  TotalCreditAmount
  1           1           2                  100000            100000
```

### Process files from your ODFI

```
$ curl -XPUT "http://localhost:9494/trigger-inbound"
achgateway_1  | ts=2021-06-18T23:39:06Z msg="starting odfi periodic processing for testing" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:39:06Z msg="start retrieving and processing of inbound files in ftp:2121" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:39:06Z msg="created directory storage-1/download318650464" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:39:06Z msg="*upload.FTPTransferAgent found 1 inbound files in /returned/" app=achgateway version=v0.4.1 level=info
achgateway_1  | ts=2021-06-18T23:39:06Z msg="saved return-WEB.ach at storage-1/download318650464/returned/return-WEB.ach" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:39:06Z msg="*upload.FTPTransferAgent found 1 reconciliation files in /reconciliation/" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:39:06Z msg="saved ppd-debit.ach at storage-1/download318650464/reconciliation/ppd-debit.ach" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:39:06Z msg="*upload.FTPTransferAgent found 1 return files in /returned/" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:39:06Z msg="saved return-WEB.ach at storage-1/download318650464/returned/return-WEB.ach" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:39:06Z msg="odfi: processing return file" app=achgateway version=v0.4.1 origin=691000134 destination=091400606 level=info
achgateway_1  | ts=2021-06-18T23:39:06Z msg="odfi: return batch 0 entry 0 code R01" app=achgateway origin=691000134 destination=091400606 version=v0.4.1 level=info
achgateway_1  | ts=2021-06-18T23:39:06Z msg="odfi: return batch 1 entry 0 code R03" version=v0.4.1 level=info app=achgateway origin=691000134 destination=091400606
achgateway_1  | ts=2021-06-18T23:39:06Z msg="cleanup: deleted remote file /returned/return-WEB.ach" version=v0.4.1 level=info app=achgateway
achgateway_1  | ts=2021-06-18T23:39:06Z msg="cleanup: deleted remote file /reconciliation/ppd-debit.ach" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:39:06Z msg="cleanup: deleted remote file /returned/return-WEB.ach" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:39:06Z msg="finished odfi periodic processing for testing" app=achgateway version=v0.4.1 level=info
```
