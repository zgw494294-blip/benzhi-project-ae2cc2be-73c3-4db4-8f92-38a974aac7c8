package core

import (
	"encoding/json"
	"time"
)

func CloneBatch(b KilnBatch) (KilnBatch, error) {
	data, err := json.Marshal(b)
	if err != nil {
		return KilnBatch{}, err
	}
	var out KilnBatch
	err = json.Unmarshal(data, &out)
	return out, err
}
func SnapshotBytes(s Snapshot) ([]byte, error) { return json.MarshalIndent(s, "", "  ") }

// cloneBatch returns a deep copy of b so that mutating the returned value
// (including slice elements, map entries and referenced structs) cannot
// reach the stored data. It keeps pointer semantics where the caller relies
// on nil-vs-set pointers while ensuring the backing memory is independent.
func cloneBatch(b KilnBatch) KilnBatch {
	out := b
	out.Stages = cloneStages(b.Stages)
	out.Readings = cloneReadings(b.Readings)
	out.Actions = cloneActions(b.Actions)
	out.PendingRetests = cloneRetests(b.PendingRetests)
	out.FrozenAt = cloneTimePtr(b.FrozenAt)
	out.ApprovedAt = cloneTimePtr(b.ApprovedAt)
	out.Certificate = cloneCertificatePtr(b.Certificate)
	return out
}

func cloneStages(in []HeatingStage) []HeatingStage {
	if in == nil {
		return nil
	}
	out := make([]HeatingStage, len(in))
	for i := range in {
		out[i] = cloneStage(in[i])
	}
	return out
}

func cloneStage(in HeatingStage) HeatingStage {
	out := in
	out.SensorIDs = cloneStrings(in.SensorIDs)
	return out
}

func cloneReadings(in []TemperatureReading) []TemperatureReading {
	if in == nil {
		return nil
	}
	out := make([]TemperatureReading, len(in))
	copy(out, in)
	return out
}

func cloneActions(in []DeviationAction) []DeviationAction {
	if in == nil {
		return nil
	}
	out := make([]DeviationAction, len(in))
	copy(out, in)
	return out
}

func cloneRetests(in map[string]RetestState) map[string]RetestState {
	if in == nil {
		return nil
	}
	out := make(map[string]RetestState, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStrings(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func cloneTimePtr(in *time.Time) *time.Time {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}

func cloneCertificatePtr(in *ReleaseCertificate) *ReleaseCertificate {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
