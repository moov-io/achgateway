---
layout: page
title: Errors
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Errors

When ACHGateway encounters an error during processing it will attempt to notify an external system. This is advised to alert a human to help resolve or monitor the situtation. ACHGateway attempts basic retry strategies in most cases, but often cannot successfully complete processing.

TODO(adam): Link to config

## PagerDuty

Critical events are triggered with some basic details of the error. Often this is an error with file uploading (network failures, invalid credentials, etc) which require human intervention.

TODO(adam): Example

## Slack

Messages are posted to a slack channel to alert humans of network or credential issues.

TODO(adam): Example
