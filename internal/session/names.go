// Package session provides polecat session lifecycle management.
package session

import (
	"fmt"
)

// DefaultPrefix is the default beads prefix used when no rig-specific prefix is known.
const DefaultPrefix = "gt"

// HQPrefix is the prefix for town-level services (Mayor, Deacon).
const HQPrefix = "hq-"

// townPrefix returns the town name prefix for session names.
// Returns "{townName}-" if a town name is set, empty string otherwise.
func townPrefix() string {
	tn := GetTownName()
	if tn == "" {
		return ""
	}
	return tn + "-"
}

// MayorSessionName returns the session name for the Mayor agent.
// With town prefix: "gt-hq-mayor", without: "hq-mayor".
func MayorSessionName() string {
	return townPrefix() + HQPrefix + "mayor"
}

// DeaconSessionName returns the session name for the Deacon agent.
// With town prefix: "gt-hq-deacon", without: "hq-deacon".
func DeaconSessionName() string {
	return townPrefix() + HQPrefix + "deacon"
}

// WitnessSessionName returns the session name for a rig's Witness agent.
// rigPrefix is the rig's beads prefix (e.g., "gt" for gastown, "bd" for beads).
// With town prefix: "gt-dma-witness", without: "dma-witness".
func WitnessSessionName(rigPrefix string) string {
	return fmt.Sprintf("%s%s-witness", townPrefix(), rigPrefix)
}

// RefinerySessionName returns the session name for a rig's Refinery agent.
// rigPrefix is the rig's beads prefix (e.g., "gt" for gastown, "bd" for beads).
// With town prefix: "gt-dma-refinery", without: "dma-refinery".
func RefinerySessionName(rigPrefix string) string {
	return fmt.Sprintf("%s%s-refinery", townPrefix(), rigPrefix)
}

// CrewSessionName returns the session name for a crew worker in a rig.
// rigPrefix is the rig's beads prefix (e.g., "gt" for gastown, "bd" for beads).
// With town prefix: "gt-dma-crew-max", without: "dma-crew-max".
func CrewSessionName(rigPrefix, name string) string {
	return fmt.Sprintf("%s%s-crew-%s", townPrefix(), rigPrefix, name)
}

// PolecatSessionName returns the session name for a polecat in a rig.
// rigPrefix is the rig's beads prefix (e.g., "gt" for gastown, "bd" for beads).
// With town prefix: "gt-dma-furiosa", without: "dma-furiosa".
func PolecatSessionName(rigPrefix, name string) string {
	return fmt.Sprintf("%s%s-%s", townPrefix(), rigPrefix, name)
}

// OverseerSessionName returns the session name for the human operator.
// The overseer is the human who controls Gas Town, not an AI agent.
// With town prefix: "gt-hq-overseer", without: "hq-overseer".
func OverseerSessionName() string {
	return townPrefix() + HQPrefix + "overseer"
}

// BootSessionName returns the session name for the Boot watchdog.
// Boot is town-level (launched by deacon), so it uses the hq- prefix.
// With town prefix: "gt-hq-boot", without: "hq-boot".
func BootSessionName() string {
	return townPrefix() + HQPrefix + "boot"
}
