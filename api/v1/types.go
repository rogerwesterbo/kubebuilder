package v1

type BackupType string

const (
	BackupTypeCronjob BackupType = "cronjob"
	BackupTypeJob     BackupType = "job"
)

type BackupPhase string

const (
	BackupPhasePending   BackupPhase = "pending"
	BackupPhaseRunning   BackupPhase = "running"
	BackupPhaseSucceeded BackupPhase = "succeeded"
	BackupPhaseFailed    BackupPhase = "failed"
)
