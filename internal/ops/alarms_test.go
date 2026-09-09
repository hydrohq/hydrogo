package ops

import (
	"testing"
	"time"
)

func TestEvaluateFailsClosedForMissingBackupAndCriticalSignals(t *testing.T) {
	alarms := Evaluate(Snapshot{Now: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), BackupMaxAge: time.Hour, DiskFreeBytes: 10, DiskMinBytes: 100, MigrationFailed: true, SMTPFailures: 2, DeadLettered: 1})
	if len(alarms) != 5 {
		t.Fatalf("alarm count = %d", len(alarms))
	}
	if alarms[0].Code != "backup_stale" || alarms[1].Code != "disk_low" || alarms[2].Code != "migration_failed" || alarms[3].Code != "smtp_delivery" || alarms[4].Code != "jobs_dead_lettered" {
		t.Fatalf("alarm order = %#v", alarms)
	}
}

func TestEvaluateReturnsNoAlarmForHealthySnapshot(t *testing.T) {
	now := time.Now().UTC()
	if alarms := Evaluate(Snapshot{Now: now, LastBackup: now.Add(-time.Minute), BackupMaxAge: time.Hour, DiskFreeBytes: 1000, DiskMinBytes: 100, SMTPFailures: 0}); len(alarms) != 0 {
		t.Fatalf("healthy alarms = %#v", alarms)
	}
}
