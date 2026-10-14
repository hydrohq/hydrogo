# ADR-0222: Backup Manifest Edition, Retention, and Encryption Boundaries

## Status

Accepted — 2026-09-22

## Decision

The SQLite and PostgreSQL backup procedures are preserved; the backup manifest optionally includes `edition`, `retention_class`, and an external KMS/key-escrow reference (`encryption_ref`). HydroGo does not generate encryption keys internally or duplicate external backup engines. These fields are captured as bounded metadata from deployment environment variables; actual encryption and physical media storage remain the responsibility of external infrastructure or operators.

## Operations

`HYDRO_BACKUP_RETENTION_CLASS` defines the statutory retention class, and `HYDRO_BACKUP_ENCRYPTION_REF` captures the KMS key identifier. Manifest verification, staging drills, and job quarantine procedures remain mandatory prior to restoring backups.
