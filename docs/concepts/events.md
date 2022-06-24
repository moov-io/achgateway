---
layout: page
title: Events
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Events

As ACHGateway uploads and retrieves files with the remote servers it will emit events. These are defined in the [`models` package](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/models) and include both Submission and ODFI events.

Events may be delivered over a HTTP webhook or supported Stream provider (e.g. Kafka). Events are encoded in their JSON format and may be optionally encrypted. To reveal events the [`compliance` package can be used](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/compliance).

TODO(adam): link to config
