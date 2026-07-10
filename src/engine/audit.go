package engine

import "solsticedtl/src/audit"

func (s *Service) Audit() audit.Report {
	return audit.Build(s.state)
}
