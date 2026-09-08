// Package ops contains deterministic operational alarm evaluation.
package ops

import "time"

type Snapshot struct {
	Now             time.Time
	LastBackup      time.Time
	BackupMaxAge    time.Duration
	DiskFreeBytes   uint64
	DiskMinBytes    uint64
	MigrationFailed bool
	SMTPFailures    int
}

type Alarm struct {
	Code     string
	Severity string
	Message  string
}

func Evaluate(s Snapshot) []Alarm {
	alarms := make([]Alarm, 0, 4)
	if s.BackupMaxAge > 0 && (s.LastBackup.IsZero() || s.Now.Sub(s.LastBackup) > s.BackupMaxAge) {
		alarms = append(alarms, Alarm{Code: "backup_stale", Severity: "critical", Message: "last successful backup is outside the allowed age"})
	}
	if s.DiskMinBytes > 0 && s.DiskFreeBytes < s.DiskMinBytes {
		alarms = append(alarms, Alarm{Code: "disk_low", Severity: "critical", Message: "free disk space is below the activation threshold"})
	}
	if s.MigrationFailed {
		alarms = append(alarms, Alarm{Code: "migration_failed", Severity: "critical", Message: "last migration did not complete successfully"})
	}
	if s.SMTPFailures > 0 {
		alarms = append(alarms, Alarm{Code: "smtp_delivery", Severity: "warning", Message: "notification delivery failures require provider review"})
	}
	return alarms
}
