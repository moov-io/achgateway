---
layout: page
title: Leadership
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Leader Elections

ACHGateway can operate in a few modes that balance scalability, availability, and network/file usage. These modes can optionally include a common distributed system task called [leader election](https://en.wikipedia.org/wiki/Leader_election) to establish which ACHGateway instance will perform the upload, while others simply update their mergable directory to simulate upload. Involing leader election in uploads is performant and does require a stable network at each cutoff. If ACHGateway instances are unable to obtain leadership or fail due to a network error the instance will not upload ACH files.

An instance of ACHGateway will upload **all files** in the shard's directory (e.g. `storage/mergable/$shard`) at a cutoff window. There's no communication between ACHGateway instances to share or merge files.

### Disabled

If ACHGateway is configured without a `Consul` block it will not perform leader election. At cutoff times it will merge and upload all files within the Shard's mergable directory of that ACHGateway instance.

### Enabled

When ACHGateway is configured with a `Consul` block it will perform leader election after merging pending files, but prior to upload.  The ACHGateway instance will attempt to elect itself for the triggered shard and upload only when it is returned as the leader. Only one instance will be the leader for a shard.

If leader election is configured then **ACHGateway instances should receive the same files** for shards. Submitting files to each instance would keep the pending files consistent across instances and any ACHGateway instance can upload them. If submitted files are not consistent across instances it can result in files not uploaded to the ODFI.

