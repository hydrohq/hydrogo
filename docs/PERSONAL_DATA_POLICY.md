# HydroGo Compact Personal Data Policy

This document defines the actionable data lifecycle for the R3 Site Edition form inbox. The form inbox is used strictly for allowlisted form identifiers; at the start of Compact, the `contact` form is supported.

## Collection and Access

- Only fields explicitly defined in the form schema are accepted; unknown fields are rejected.
- Form submissions are protected by rate limits and payload size constraints.
- Inbox records can only be read and have their status modified by the admin role. Editor and viewer roles cannot access the personal data inbox.
- Audit records store the operation type and target identifier, not the personal data payload itself.

## Retention, Erasure, and Export

- Inbox records are retained for 30 days by default; `PurgeInbox` purges only expired records.
- Customer-requested access and export are executed manually via a runbook step following verified user request by an administrator; export files are placed in encrypted temporary storage and removed immediately after delivery.
- For erasure requests, the inbox record and associated media targets are appended as an idempotent intent to the external erasure log; the record is not deemed erased until log replay completes.
- Records in backups remain encrypted until backup retention expires; following a restore, the same purge and erasure replay steps must be re-executed.

## Operational Evidence

Every access, export, status transition, and purge action is bound to an audit event. The operator records the checkpoint, count of affected records, and any replay failures into the operational runbook.
