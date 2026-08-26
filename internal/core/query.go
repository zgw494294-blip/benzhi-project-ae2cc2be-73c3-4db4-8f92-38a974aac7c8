package core

func (s *Store) Counts() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := map[string]int{StatusDraft: 0, StatusCollect: 0, StatusReview: 0, StatusApproved: 0, StatusRejected: 0}
	for _, b := range s.batches {
		out[b.Status]++
	}
	return out
}
func (s *Store) Stage(id, stageID string) (HeatingStage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.batches[id]
	if !ok {
		return HeatingStage{}, false
	}
	for _, st := range b.Stages {
		if st.ID == stageID {
			return st, true
		}
	}
	return HeatingStage{}, false
}
