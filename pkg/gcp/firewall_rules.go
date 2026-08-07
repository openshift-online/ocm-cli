package gcp

import (
	"fmt"
	"strings"

	cmv1 "github.com/openshift-online/ocm-api-model/clientapi/clustersmgmt/v1"
)

type FirewallRuleset struct {
	*cmv1.GcpFirewallRule
}

func NewFirewallRuleset(ruleset *cmv1.GcpFirewallRule) *FirewallRuleset {
	return &FirewallRuleset{
		ruleset,
	}
}

// GenerateCreateBashScript generates a bash script with all firewall rule commands
// Used for manual mode, where the commands are not called during runtime
func GenerateCreateBashScript(fr *FirewallRuleset) string {
	var sb strings.Builder

	sb.WriteString("#!/bin/bash\n\n")
	sb.WriteString("# GCP Firewall Rules Create Script\n")
	sb.WriteString("# Generated automatically\n\n")
	fmt.Fprintf(&sb, "PROJECT=%s\n", shellQuote(fr.GcpNetwork().ProjectId()))
	fmt.Fprintf(&sb, "NETWORK=%s\n", shellQuote(fr.GcpNetwork().VpcName()))
	fmt.Fprintf(&sb, "PREFIX=%s\n\n", shellQuote(fr.Name()))

	sb.WriteString("set -e\n\n")

	for i, rule := range fr.Status().Rules() {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "# %s\n", shellComment(rule.Name()))
		sb.WriteString(generateCreateCommand(rule, fr.GcpNetwork().ProjectId()))
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateCreateCommand generates the gcloud command for a firewall rule
func generateCreateCommand(
	rule *cmv1.GcpFirewallRuleSpec,
	project string,
) string {
	cmd := fmt.Sprintf("gcloud compute --project=%s firewall-rules create %s",
		shellQuote(project), shellQuote(rule.Name()))

	if rule.Direction() != "" {
		cmd += fmt.Sprintf(" --direction=%s", shellQuote(rule.Direction()))
	}

	if priority, ok := rule.GetPriority(); ok {
		cmd += fmt.Sprintf(" --priority=%d", priority)
	}

	cmd += fmt.Sprintf(" --network=%s", shellQuote(rule.Network()))
	cmd += " --action=ALLOW"

	// Convert allowed protocols and ports to --rules format
	if len(rule.Allowed()) > 0 {
		rules := buildRulesString(rule.Allowed())
		if rules != "" {
			cmd += fmt.Sprintf(" --rules=%s", shellQuote(rules))
		}
	}

	// Use raw fields extracted from JSON
	if len(rule.SourceRanges()) > 0 {
		cmd += fmt.Sprintf(" --source-ranges=%s", shellQuote(strings.Join(rule.SourceRanges(), ",")))
	}
	if len(rule.SourceServiceAccounts()) > 0 {
		cmd += fmt.Sprintf(" --source-service-accounts=%s", shellQuote(strings.Join(rule.SourceServiceAccounts(), ",")))
	}
	if len(rule.TargetServiceAccounts()) > 0 {
		cmd += fmt.Sprintf(" --target-service-accounts=%s", shellQuote(strings.Join(rule.TargetServiceAccounts(), ",")))
	}

	cmd += " --quiet"

	return cmd
}

// GenerateDeleteBashScript generates a bash script with all firewall rule commands
// Used for manual mode, where the commands are not called during runtime
func GenerateDeleteBashScript(fr *FirewallRuleset) string {
	var sb strings.Builder

	sb.WriteString("#!/bin/bash\n\n")
	sb.WriteString("# GCP Firewall Rules Delete Script\n")
	sb.WriteString("# Generated automatically\n\n")
	fmt.Fprintf(&sb, "PROJECT=%s\n", shellQuote(fr.GcpNetwork().ProjectId()))
	fmt.Fprintf(&sb, "NETWORK=%s\n", shellQuote(fr.GcpNetwork().VpcName()))
	fmt.Fprintf(&sb, "PREFIX=%s\n\n", shellQuote(fr.Name()))

	sb.WriteString("set -e\n\n")

	for i, rule := range fr.Status().Rules() {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "# %s\n", shellComment(rule.Name()))
		sb.WriteString(generateDeleteCommand(rule, fr.GcpNetwork().ProjectId()))
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateDeleteCommand generates the gcloud command for deleting a firewall rule
func generateDeleteCommand(
	rule *cmv1.GcpFirewallRuleSpec,
	project string,
) string {
	cmd := fmt.Sprintf("gcloud compute --project=%s firewall-rules delete %s --quiet",
		shellQuote(project), shellQuote(rule.Name()))
	return cmd
}

// GenerateUpdateBashScript generates a bash script with all firewall rule update commands
// Used for manual mode, where the commands ensure the firewall rules match the intended configuration
func GenerateUpdateBashScript(fr *FirewallRuleset) string {
	var sb strings.Builder

	sb.WriteString("#!/bin/bash\n\n")
	sb.WriteString("# GCP Firewall Rules Update Script\n")
	sb.WriteString("# Generated automatically\n")
	sb.WriteString("# This script ensures firewall rules match the intended configuration\n\n")
	fmt.Fprintf(&sb, "PROJECT=%s\n", shellQuote(fr.GcpNetwork().ProjectId()))
	fmt.Fprintf(&sb, "NETWORK=%s\n", shellQuote(fr.GcpNetwork().VpcName()))
	fmt.Fprintf(&sb, "PREFIX=%s\n\n", shellQuote(fr.Name()))

	sb.WriteString("set -e\n\n")

	for i, rule := range fr.Status().Rules() {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "# Update %s\n", shellComment(rule.Name()))
		sb.WriteString(generateUpdateCommand(rule, fr.GcpNetwork().ProjectId()))
		sb.WriteString("\n")
	}

	sb.WriteString("\necho 'Firewall rules updated successfully'\n")

	return sb.String()
}

// generateUpdateCommand generates the gcloud update command for a firewall rule
// Note: direction and network cannot be updated - those require delete + recreate
func generateUpdateCommand(
	rule *cmv1.GcpFirewallRuleSpec,
	project string,
) string {
	cmd := fmt.Sprintf("gcloud compute --project=%s firewall-rules update %s",
		shellQuote(project), shellQuote(rule.Name()))

	if priority, ok := rule.GetPriority(); ok {
		cmd += fmt.Sprintf(" --priority=%d", priority)
	}

	// Convert allowed protocols and ports to --rules format
	if len(rule.Allowed()) > 0 {
		rules := buildRulesString(rule.Allowed())
		if rules != "" {
			cmd += fmt.Sprintf(" --rules=%s", shellQuote(rules))
		}
	}

	// Source and target configurations
	// Always emit these flags (even when empty) to ensure clean state
	if len(rule.SourceRanges()) > 0 {
		cmd += fmt.Sprintf(" --source-ranges=%s", shellQuote(strings.Join(rule.SourceRanges(), ",")))
	} else {
		cmd += " --source-ranges="
	}
	if len(rule.SourceServiceAccounts()) > 0 {
		cmd += fmt.Sprintf(" --source-service-accounts=%s", shellQuote(strings.Join(rule.SourceServiceAccounts(), ",")))
	} else {
		cmd += " --source-service-accounts="
	}
	if len(rule.TargetServiceAccounts()) > 0 {
		cmd += fmt.Sprintf(" --target-service-accounts=%s", shellQuote(strings.Join(rule.TargetServiceAccounts(), ",")))
	} else {
		cmd += " --target-service-accounts="
	}

	// Clear source/target tags (to ensure clean state)
	cmd += " --source-tags="
	cmd += " --target-tags="
	cmd += " --quiet"

	return cmd
}

// buildRulesString converts the Allowed array to the gcloud --rules format
// e.g., "tcp:22,tcp:80,udp:53,icmp"
func buildRulesString(
	allowList []*cmv1.GcpFirewallRuleAllowed,
) string {
	var parts []string

	for _, allowed := range allowList {
		protocol := allowed.IPProtocol()

		if len(allowed.Ports()) == 0 {
			// Protocol only (e.g., "icmp", "esp")
			parts = append(parts, protocol)
		} else {
			// Protocol with ports (e.g., "tcp:22,tcp:80")
			for _, port := range allowed.Ports() {
				parts = append(parts, fmt.Sprintf("%s:%s", protocol, port))
			}
		}
	}

	return strings.Join(parts, ",")
}

// shellQuote escapes a string for safe use in POSIX shell commands.
// It wraps the string in single quotes and escapes any embedded single quotes.
func shellQuote(s string) string {
	// Replace each single quote with '\'' (end quote, escaped quote, start quote)
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return fmt.Sprintf("'%s'", escaped)
}

// shellComment sanitizes input by removing all carriage returns and newlines.
func shellComment(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.ReplaceAll(s, "\n", " ")
}
