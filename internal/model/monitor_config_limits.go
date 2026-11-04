package model

// Monitor config raw keys are UTF-8 strings, never normalized or truncated.
// The binary namespace/journal columns reserve four bytes per Unicode code point.
const (
	MonitorConfigKeyMaxRunes  = 128
	MonitorConfigKeyMaxBytes  = 4 * MonitorConfigKeyMaxRunes
	MonitorConfigNameMaxRunes = 255
)
