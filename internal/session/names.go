// Package session provides polecat session lifecycle management.
package session

import (
	"fmt"
	"sync"
)

// DefaultPrefix is the default beads prefix used when no rig-specific prefix is known.
const DefaultPrefix = "gt"

// hqPrefix is the prefix for town-level services (Mayor, Deacon).
// Initialized by InitHQPrefix during registry setup; defaults to "hq-"
// for backward compatibility with single-town deployments.
var hqPrefix = "hq-"
var hqPrefixMu sync.RWMutex

// HQPrefix returns the current HQ prefix for town-level session names.
// Thread-safe for concurrent access.
func HQPrefix() string {
	hqPrefixMu.RLock()
	defer hqPrefixMu.RUnlock()
	return hqPrefix
}

// SetHQPrefix sets the HQ prefix for town-level session names.
// Called during InitRegistry with the town name from town.json.
// Format: "<town-name>-hq-" (e.g., "gt-hq-", "paper-town-hq-").
func SetHQPrefix(prefix string) {
	hqPrefixMu.Lock()
	defer hqPrefixMu.Unlock()
	hqPrefix = prefix
}

// InitHQPrefixFromTownName derives and sets the HQ prefix from the town name.
// The prefix format is "<sanitized-town-name>-hq-" to namespace town-level
// sessions per town, preventing collisions when multiple towns share a host.
func InitHQPrefixFromTownName(townName string) {
	sanitized := sanitizeTownName(townName)
	SetHQPrefix(sanitized + "-hq-")
}

// MayorSessionName returns the session name for the Mayor agent.
// Session name is namespaced by town (e.g., "gt-hq-mayor", "paper-town-hq-mayor").
func MayorSessionName() string {
	return HQPrefix() + "mayor"
}

// DeaconSessionName returns the session name for the Deacon agent.
// Session name is namespaced by town (e.g., "gt-hq-deacon", "paper-town-hq-deacon").
func DeaconSessionName() string {
	return HQPrefix() + "deacon"
}

// WitnessSessionName returns the session name for a rig's Witness agent.
// rigPrefix is the rig's beads prefix (e.g., "gt" for gastown, "bd" for beads).
func WitnessSessionName(rigPrefix string) string {
	return fmt.Sprintf("%s-witness", rigPrefix)
}

// RefinerySessionName returns the session name for a rig's Refinery agent.
// rigPrefix is the rig's beads prefix (e.g., "gt" for gastown, "bd" for beads).
func RefinerySessionName(rigPrefix string) string {
	return fmt.Sprintf("%s-refinery", rigPrefix)
}

// CrewSessionName returns the session name for a crew worker in a rig.
// rigPrefix is the rig's beads prefix (e.g., "gt" for gastown, "bd" for beads).
func CrewSessionName(rigPrefix, name string) string {
	return fmt.Sprintf("%s-crew-%s", rigPrefix, name)
}

// PolecatSessionName returns the session name for a polecat in a rig.
// rigPrefix is the rig's beads prefix (e.g., "gt" for gastown, "bd" for beads).
func PolecatSessionName(rigPrefix, name string) string {
	return fmt.Sprintf("%s-%s", rigPrefix, name)
}

// OverseerSessionName returns the session name for the human operator.
// The overseer is the human who controls Gas Town, not an AI agent.
func OverseerSessionName() string {
	return HQPrefix() + "overseer"
}

// BootSessionName returns the session name for the Boot watchdog.
// Boot is town-level (launched by deacon), so it uses the town-namespaced hq- prefix.
func BootSessionName() string {
	return HQPrefix() + "boot"
}
