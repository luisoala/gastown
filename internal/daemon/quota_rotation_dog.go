package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const defaultQuotaRotationDogInterval = 3 * time.Minute

// QuotaRotationDogConfig holds configuration for the quota_rotation_dog patrol.
type QuotaRotationDogConfig struct {
	// Enabled controls whether the quota rotation dog runs.
	Enabled bool `json:"enabled"`

	// IntervalStr is how often to run, as a string (e.g., "3m").
	IntervalStr string `json:"interval,omitempty"`

	// IdleOnly restricts rotation to idle sessions only.
	// Default: true (only rotate when the active session is idle).
	IdleOnly *bool `json:"idle_only,omitempty"`
}

// quotaRotationDogInterval returns the configured interval, or the default (3m).
func quotaRotationDogInterval(config *DaemonPatrolConfig) time.Duration {
	if config != nil && config.Patrols != nil && config.Patrols.QuotaRotationDog != nil {
		if config.Patrols.QuotaRotationDog.IntervalStr != "" {
			if d, err := time.ParseDuration(config.Patrols.QuotaRotationDog.IntervalStr); err == nil && d > 0 {
				return d
			}
		}
	}
	return defaultQuotaRotationDogInterval
}

// quotaRotationReport holds the result of a scan+rotate cycle.
type quotaRotationReport struct {
	Timestamp    time.Time `json:"timestamp"`
	ScanResult   string    `json:"scan_result,omitempty"`
	ScanError    string    `json:"scan_error,omitempty"`
	RotateResult string    `json:"rotate_result,omitempty"`
	RotateError  string    `json:"rotate_error,omitempty"`
}

// runQuotaRotationDog scans quota status and rotates if needed.
func (d *Daemon) runQuotaRotationDog() {
	if !IsPatrolEnabled(d.patrolConfig, "quota_rotation_dog") {
		return
	}

	d.logger.Printf("quota_rotation_dog: starting scan+rotate cycle")

	report := &quotaRotationReport{
		Timestamp: time.Now(),
	}

	// Step 1: Scan quota status
	scanCtx, scanCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer scanCancel()

	gtBin := d.gtPath
	if gtBin == "" {
		gtBin = "gt"
	}

	scanCmd := exec.CommandContext(scanCtx, gtBin, "quota", "scan", "--update", "--json")
	scanOut, scanErr := scanCmd.CombinedOutput()
	if scanErr != nil {
		report.ScanError = fmt.Sprintf("%v: %s", scanErr, string(scanOut))
		d.logger.Printf("quota_rotation_dog: scan failed: %s", report.ScanError)
	} else {
		report.ScanResult = string(scanOut)
		d.logger.Printf("quota_rotation_dog: scan complete")
	}

	// Step 2: Rotate if scan succeeded
	if scanErr == nil {
		rotateCtx, rotateCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer rotateCancel()

		rotateArgs := []string{"quota", "rotate", "--json"}

		// Check idle_only config (default true)
		idleOnly := true
		if d.patrolConfig != nil && d.patrolConfig.Patrols != nil &&
			d.patrolConfig.Patrols.QuotaRotationDog != nil &&
			d.patrolConfig.Patrols.QuotaRotationDog.IdleOnly != nil {
			idleOnly = *d.patrolConfig.Patrols.QuotaRotationDog.IdleOnly
		}
		if idleOnly {
			rotateArgs = append(rotateArgs, "--idle")
		}

		rotateCmd := exec.CommandContext(rotateCtx, gtBin, rotateArgs...)
		rotateOut, rotateErr := rotateCmd.CombinedOutput()
		if rotateErr != nil {
			report.RotateError = fmt.Sprintf("%v: %s", rotateErr, string(rotateOut))
			d.logger.Printf("quota_rotation_dog: rotate failed: %s", report.RotateError)

			// Escalate on persistent rotation failures
			d.escalate("quota_rotation_dog", fmt.Sprintf("rotation failed: %s", report.RotateError))
		} else {
			report.RotateResult = string(rotateOut)
			d.logger.Printf("quota_rotation_dog: rotate complete")
		}
	}

	// Write report for debugging
	d.writeQuotaRotationReport(report)

	d.logger.Printf("quota_rotation_dog: cycle complete")
}

// writeQuotaRotationReport writes the report as JSON to the town root.
func (d *Daemon) writeQuotaRotationReport(report *quotaRotationReport) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		d.logger.Printf("quota_rotation_dog: failed to marshal report: %v", err)
		return
	}

	reportPath := fmt.Sprintf("%s/.quota-rotation-report.json", d.config.TownRoot)
	if err := writeFileAtomic(reportPath, data, 0644); err != nil {
		d.logger.Printf("quota_rotation_dog: failed to write report: %v", err)
	}
}

// writeFileAtomic writes data to a file atomically via temp+rename.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
