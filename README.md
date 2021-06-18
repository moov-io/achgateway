<!--generated-from:11badeae7f5171e6ec312a610718a7a4ac276e18df06d4d715e771702f50aba8 DO NOT REMOVE, DO UPDATE -->
moov-io/achgateway
===

[![GoDoc](https://godoc.org/github.com/moov-io/achgateway?status.svg)](https://godoc.org/github.com/moov-io/achgateway)
[![Build Status](https://github.com/moov-io/achgateway/workflows/Go/badge.svg)](https://github.com/moov-io/achgateway/actions)
[![Coverage Status](https://codecov.io/gh/moov-io/achgateway/branch/master/graph/badge.svg)](https://codecov.io/gh/moov-io/achgateway)
[![Go Report Card](https://goreportcard.com/badge/github.com/moov-io/achgateway)](https://goreportcard.com/report/github.com/moov-io/achgateway)
[![Apache 2 licensed](https://img.shields.io/badge/license-Apache2-blue.svg)](https://raw.githubusercontent.com/moov-io/achgateway/master/LICENSE)

An extensible, highly available, distributed, and fault tolerant ACH uploader and downloader.
ACH Gateway creates events for outside services and transforms files prior to upload to fit real-world
requirements of production systems.

Docs: [docs](https://moov-io.github.io/achgateway/) | [open api specification](api/api.yml)

## Getting Started

Read through the [project docs](docs/README.md) over here to get an understanding of the purpose of this project and how to run it.

We publish a [public Docker image `moov/achgateway`](https://hub.docker.com/r/moov/achgateway/) from Docker Hub or use this repository. No configuration is required to serve on `:8484` and metrics at `:9494/metrics` in Prometheus format.

Start achgateway and an FTP server:
```
$ cd ./examples/getting-started/

$ docker-compose up
...
achgateway_1  | ts=2021-06-18T23:38:06Z msg="public listening on :8484" version=v0.4.1 level=info app=achgateway
achgateway_1  | ts=2021-06-18T23:38:06Z msg="listening on [::]:9494" version=v0.4.1 level=info app=achgateway
```

Submit a file to achgateway:
```
$ curl -XPOST "http://localhost:8484/shards/foo/files/f4" --data @./testdata/ppd-valid.json
...
achgateway_1  | ts=2021-06-18T23:38:16Z msg="begin handle received ACHFile=f4 of 1918 bytes" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:38:16Z msg="finished handling ACHFile=f4" level=info app=achgateway version=v0.4.1
```

Initiate cutoff time processing (aka upload to your ODFI):
```
$ curl -XPUT "http://localhost:9494/trigger-cutoff"
achgateway_1  | ts=2021-06-18T23:38:20Z msg="starting manual cutoff window processing" level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:38:20Z msg="processing mergable directory foo" level=info app=achgateway version=v0.4.1 shardKey=foo
achgateway_1  | ts=2021-06-18T23:38:20Z msg="found *upload.FTPTransferAgent agent" version=v0.4.1 shardKey=foo level=info app=achgateway
achgateway_1  | ts=2021-06-18T23:38:20Z msg="found 1 matching ACH files: []string{\"storage-1/20210618-233820/foo/f4.ach\"}" tenantID=foo level=info app=achgateway version=v0.4.1
achgateway_1  | ts=2021-06-18T23:38:20Z msg="merged 1 files into 1 files" level=info app=achgateway version=v0.4.1 tenantID=foo
```

View the uploaded file:
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

__Optional__ Initiate inbound file processing:
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

## Usage

achgateway accepts files over HTTP and kafka to queue them up for upload at a Nacha cutoff time. This allows systems and humans to publish files and have them be optimized for upload. achgateway is inspired from [the work done in moov-io/paygate](https://github.com/moov-io/paygate) and is used in production at Moov.

## Project Status

This project is currently under development and could introduce breaking changes to reach a stable status. Currently it runs in production. We are looking for community feedback so please try out our code or give us feedback!

## Getting Help

 channel | info
 ------- | -------
[Project Documentation](./docs/README.md) | Our project documentation available online.
Twitter [@moov_io](https://twitter.com/moov_io)	| You can follow Moov.IO's Twitter feed to get updates on our project(s). You can also tweet us questions or just share blogs or stories.
[GitHub Issue](https://github.com/moov-io/achgateway/issues) | If you are able to reproduce a problem please open a GitHub Issue under the specific project that caused the error.
[moov-io slack](https://slack.moov.io/) | Join our slack channel (`#ach`) to have an interactive discussion about the development of the project.

## Supported and Tested Platforms

- 64-bit Linux (Ubuntu, Debian), macOS, and Windows

## Contributing

Yes please! Please review our [Contributing guide](CONTRIBUTING.md) and [Code of Conduct](https://github.com/moov-io/ach/blob/master/CODE_OF_CONDUCT.md) to get started! Checkout our [issues for first time contributors](https://github.com/moov-io/achgateway/contribute) for something to help out with.

This project uses [Go Modules](https://github.com/golang/go/wiki/Modules) and uses Go 1.14 or higher. See [Golang's install instructions](https://golang.org/doc/install) for help setting up Go. You can download the source code and we offer [tagged and released versions](https://github.com/moov-io/achgateway/releases/latest) as well. We highly recommend you use a tagged release for production.

### Test Coverage

Improving test coverage is a good candidate for new contributors while also allowing the project to move more quickly by reducing regressions issues that might not be caught before a release is pushed out to our users. One great way to improve coverage is by adding edge cases and different inputs to functions (or [contributing and running fuzzers](https://github.com/dvyukov/go-fuzz)).

Tests can run processes (like sqlite databases), but should only do so locally.

## License

Apache License 2.0 See [LICENSE](LICENSE) for details.
