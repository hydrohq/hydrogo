package ops

import (
	"testing"
	"time"
)

func TestEvaluateFailsClosedForMissingBackupAndCriticalSignals(t *testing.T) {
	alarms := Evaluate(Snapshot{Now: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), BackupMaxAge: time.Hour, DiskFreeBytes: 10, DiskMinBytes: 100, MigrationFailed: true, ReadinessFailed: true, SMTPFailures: 2, DeadLettered: 1})
	if len(alarms) != 6 {
		t.Fatalf("alarm count = %d", len(alarms))
	}
	if alarms[0].Code != "backup_stale" || alarms[1].Code != "disk_low" || alarms[2].Code != "migration_failed" || alarms[3].Code != "readiness_failed" || alarms[4].Code != "smtp_delivery" || alarms[5].Code != "jobs_dead_lettered" {
		t.Fatalf("alarm order = %#v", alarms)
	}
}

func TestEvaluateReturnsNoAlarmForHealthySnapshot(t *testing.T) {
	now := time.Now().UTC()
	if alarms := Evaluate(Snapshot{Now: now, LastBackup: now.Add(-time.Minute), BackupMaxAge: time.Hour, DiskFreeBytes: 1000, DiskMinBytes: 100, SMTPFailures: 0}); len(alarms) != 0 {
		t.Fatalf("healthy alarms = %#v", alarms)
	}
}

func TestEvaluateClearsReadinessAlarmAfterRecovery(t *testing.T) {
	now := time.Now().UTC()
	unhealthy := Snapshot{Now: now, LastBackup: now, BackupMaxAge: time.Hour, DiskFreeBytes: 1000, DiskMinBytes: 100, ReadinessFailed: true}
	alarms := Evaluate(unhealthy)
	if len(alarms) != 1 || alarms[0].Code != "readiness_failed" {
		t.Fatalf("unhealthy readiness alarms = %#v", alarms)
	}

	unhealthy.ReadinessFailed = false
	if alarms := Evaluate(unhealthy); len(alarms) != 0 {
		t.Fatalf("recovered readiness alarms = %#v", alarms)
	}
}
