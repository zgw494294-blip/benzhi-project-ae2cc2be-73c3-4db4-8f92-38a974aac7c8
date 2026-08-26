package validation

func (r StageReport) Explanation() string {
	if r.Compliant {
		return "阶段所有测点均满足速率、目标温度和保温窗口"
	}
	if len(r.Findings) == 0 {
		return "阶段未通过但没有可解释发现"
	}
	return r.Findings[0].Message
}
func (r BatchReport) FailedStages() []string {
	out := []string{}
	for _, s := range r.Stages {
		if !s.Compliant {
			out = append(out, s.StageID)
		}
	}
	return out
}
