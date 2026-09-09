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
	ReadinessFailed bool
	SMTPFailures    int
	DeadLettered    int64
}

type Alarm struct {
	Code     string
	Severity string
	Message  string
}

func Evaluate(s Snapshot) []Alarm {
	alarms := make([]Alarm, 0, 6)
	if s.BackupMaxAge > 0 && (s.LastBackup.IsZero() || s.Now.Sub(s.LastBackup) > s.BackupMaxAge) {
		alarms = append(alarms, Alarm{Code: "backup_stale", Severity: "critical", Message: "last successful backup is outside the allowed age"})
	}
	if s.DiskMinBytes > 0 && s.DiskFreeBytes < s.DiskMinBytes {
		alarms = append(alarms, Alarm{Code: "disk_low", Severity: "critical", Message: "free disk space is below the activation threshold"})
	}
	if s.MigrationFailed {
		alarms = append(alarms, Alarm{Code: "migration_failed", Severity: "critical", Message: "last migration did not complete successfully"})
	}
	if s.ReadinessFailed {
		alarms = append(alarms, Alarm{Code: "readiness_failed", Severity: "critical", Message: "readiness checks are failing and the instance is not safe to receive traffic"})
	}
	if s.SMTPFailures > 0 {
		alarms = append(alarms, Alarm{Code: "smtp_delivery", Severity: "warning", Message: "notification delivery failures require provider review"})
	}
	if s.DeadLettered > 0 {
		alarms = append(alarms, Alarm{Code: "jobs_dead_lettered", Severity: "warning", Message: "dead-lettered jobs require operator review or replay"})
	}
	return alarms
}
