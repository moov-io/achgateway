---
layout: page
title: Errors
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Errors

When ACHGateway encounters processing errors, it proactively notifies designated external systems. This mechanism is crucial for engaging human intervention for resolution or monitoring. While ACHGateway implements fundamental retry strategies for most errors, some issues may persist beyond automated recovery efforts.

**Related Configuration**: Explore setting up [`Errors` notifications](../../config/#error-alerting) for detailed alert management.

## PagerDuty

For critical incidents, PagerDuty alerts are generated, providing essential error details. Common triggers include issues related to file uploads, such as network failures or incorrect credentials, which typically necessitate manual intervention.

## Slack

Slack channels receive notifications about problems like network disruptions or credential verification failures. These alerts aim to promptly inform team members, allowing for swift action to address the underlying issues.
