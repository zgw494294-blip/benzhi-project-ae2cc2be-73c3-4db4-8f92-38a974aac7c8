package core

import "encoding/json"

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
