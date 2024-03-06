---
layout: page
title: Project Goals
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Project Goals

ACHGateway streamlines the process of handling Automated Clearing House (ACH) transactions by automating the upload and download of ACH files formatted according to Nacha standards. Designed to facilitate secure and efficient transactions, this service supports FTP/SFTP server interactions and is engineered to accept and optimize valid Nacha files for seamless integration with an Originating Depository Financial Institution (ODFI).

## Key Features

ACHGateway is built with a focus on flexibility, efficiency, and security, offering a range of features designed to enhance the ACH file handling process:

- **Extensible Submissions:** Supports the submission of ACH files, including partial requests, for timely uploads according to cutoff schedules.
- **File Merging:** Combines pending files to minimize network bandwidth and optimize costs, aligning with specific time requirements throughout the day.
- **Custom Filename Templating:** Offers the ability to customize filenames for uploads, including support for handling non-compliant Nacha files.
- **Comprehensive Audit Trail:** Maintains a secure storage of all uploaded and downloaded files, with options for [retrieval or viewing](https://github.com/moov-io/ach-web-viewer), ensuring transparency and ease of access for auditing purposes.
- **Notifications:** Provides real-time alerts for successful uploads or error detections through various channels including Slack, PagerDuty, and email, ensuring stakeholders are promptly informed.

## Non-Goals

While ACHGateway excels in facilitating ACH file transactions, it is important to note the project's defined scope. The following are outside of ACHGateway's direct functionality but are essential considerations for a comprehensive ACH processing ecosystem:

- **Nacha Compliant Limit Analysis:** Assessing transaction limits as per Nacha guidelines.
- **Balance Verification:** Confirming account balances prior to transaction processing.
- **Transaction Authorization:** Securing approvals for transaction execution.
- **Settlement Availability:** Ensuring funds are available for settlement.
- **Risk Calculations:** Performing risk assessments for transactions. For related services, such as OFAC, sanction, and watchlist scanning, refer to [Watchman](https://github.com/moov-io/watchman).

## Conclusion

ACHGateway aims to enhance the efficiency and security of ACH file processing through automation, customization, and robust auditing capabilities. While addressing core needs in ACH file management, it also delineates its scope to focus on delivering optimized solutions within its designated functionalities.
