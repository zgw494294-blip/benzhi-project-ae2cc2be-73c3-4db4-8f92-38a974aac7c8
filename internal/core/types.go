package core

import "time"

const (
	StatusDraft    = "draft"
	StatusCollect  = "collecting"
	StatusReview   = "review"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

type KilnBatch struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	KilnCode       string                 `json:"kilnCode"`
	LiningMaterial string                 `json:"liningMaterial"`
	Operator       string                 `json:"operator"`
	Engineer       string                 `json:"engineer"`
	Status         string                 `json:"status"`
	PlanVersion    int                    `json:"planVersion"`
	Version        int                    `json:"version"`
	CreatedAt      time.Time              `json:"createdAt"`
	FrozenAt       *time.Time             `json:"frozenAt,omitempty"`
	ApprovedAt     *time.Time             `json:"approvedAt,omitempty"`
	Stages         []HeatingStage         `json:"stages"`
	Readings       []TemperatureReading   `json:"readings"`
	Actions        []DeviationAction      `json:"actions"`
	Certificate    *ReleaseCertificate    `json:"certificate,omitempty"`
	PendingRetests map[string]RetestState `json:"pendingRetests,omitempty"`
}

type RetestState struct {
	StageID       string    `json:"stageID"`
	ActionID      string    `json:"actionID"`
	RequiredAfter time.Time `json:"requiredAfter"`
	Required      bool      `json:"required"`
}

type HeatingStage struct {
	ID                    string   `json:"id"`
	BatchID               string   `json:"batchId"`
	Sequence              int      `json:"sequence"`
	TargetCelsius         float64  `json:"targetCelsius"`
	MaxRampCelsiusPerHour float64  `json:"maxRampCelsiusPerHour"`
	HoldMinutes           int      `json:"holdMinutes"`
	ToleranceCelsius      float64  `json:"toleranceCelsius"`
	SensorIDs             []string `json:"sensorIDs"`
	Frozen                bool     `json:"frozen"`
}

type TemperatureReading struct {
	ID         string    `json:"id"`
	BatchID    string    `json:"batchId"`
	StageID    string    `json:"stageID"`
	SensorID   string    `json:"sensorID"`
	RecordedAt time.Time `json:"recordedAt"`
	Celsius    float64   `json:"celsius"`
	Source     string    `json:"source"`
	Sequence   int       `json:"sequence"`
}

type DeviationAction struct {
	ID             string    `json:"id"`
	BatchID        string    `json:"batchId"`
	StageID        string    `json:"stageID"`
	Kind           string    `json:"kind"`
	Reason         string    `json:"reason"`
	ActionText     string    `json:"actionText"`
	PerformedBy    string    `json:"performedBy"`
	PerformedAt    time.Time `json:"performedAt"`
	RetestRequired bool      `json:"retestRequired"`
}

type ReleaseCertificate struct {
	ID            string    `json:"id"`
	BatchID       string    `json:"batchId"`
	BatchVersion  int       `json:"batchVersion"`
	Decision      string    `json:"decision"`
	Reviewer      string    `json:"reviewer"`
	ReviewComment string    `json:"reviewComment"`
	Digest        string    `json:"digest"`
	IssuedAt      time.Time `json:"issuedAt"`
	Snapshot      string    `json:"snapshot"`
}

type Snapshot struct {
	SchemaVersion int         `json:"schemaVersion"`
	SavedAt       time.Time   `json:"savedAt"`
	Batches       []KilnBatch `json:"batches"`
}
