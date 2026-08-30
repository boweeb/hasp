package app

// Check runs every detector against m and returns every Finding — advisory, non-suppressible,
// reported on managed and unmanaged resources alike (design.md §6.2). Host detectors arrive in
// Stage 15; for now this covers key and profile findings.
func Check(m Machine) []Finding {
	var findings []Finding
	findings = append(findings, checkKeys(m.Keys)...)
	findings = append(findings, checkProfiles(m.Profiles, m.Keys)...)
	return findings
}
