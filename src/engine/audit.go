package engine

import "solsticedtl/src/audit"

func (s *Service) Audit() (audit.Report, error) {
	report := audit.Build(s.state)
	schedule, err := audit.BuildSchedule(s.state, s.policy)
	if err != nil {
		return audit.Report{}, err
	}
	report.Schedule = schedule
	return report, nil
}
