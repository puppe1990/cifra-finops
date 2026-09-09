package awsinv

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCloudShellCommand_isOneLineAndEmbedsPolicy(t *testing.T) {
	cmd := CloudShellCommand()
	if cmd == "" {
		t.Fatal("empty command")
	}
	if strings.Contains(cmd, "\n") {
		t.Fatal("CloudShell command must be a single line")
	}
	for _, needle := range []string{
		"ce:GetCostAndUsage",
		"lightsail:GetInstances",
		"put-user-policy",
		"put-role-policy",
		"CifraFinOpsRead",
		"aws sts get-caller-identity",
	} {
		if !strings.Contains(cmd, needle) {
			t.Errorf("missing %q in %s", needle, cmd)
		}
	}

	doc := extractQuotedDOC(t, cmd)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(doc), &parsed); err != nil {
		t.Fatalf("DOC is not JSON: %v (%s)", err, doc)
	}
	var canonical map[string]any
	if err := json.Unmarshal([]byte(FinOpsIAMPolicy), &canonical); err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(parsed)
	want, _ := json.Marshal(canonical)
	if string(got) != string(want) {
		t.Fatalf("embedded policy != FinOpsIAMPolicy\ngot  %s\nwant %s", got, want)
	}
}

func TestCloudShellCommand_handlesAccountRoot(t *testing.T) {
	cmd := CloudShellCommand()
	if !strings.Contains(cmd, "*:root)") {
		t.Fatal("must match arn:aws:iam::ACCOUNT:root")
	}
	if !strings.Contains(cmd, "create-user") {
		t.Fatal("root cannot receive IAM policies; expected create-user")
	}
	if !strings.Contains(cmd, "cifra-finops") {
		t.Fatal("missing dedicated IAM user cifra-finops")
	}
	if !strings.Contains(cmd, "create-access-key") {
		t.Fatal("root path should emit access keys for Cifra")
	}
	if !strings.Contains(cmd, "list-access-keys") {
		t.Fatal("root path must list existing keys when create-access-key hits the 2-key quota")
	}
	if !strings.Contains(cmd, "delete-access-key") {
		t.Fatal("root path must show how to free a key slot")
	}
	if !strings.Contains(cmd, "Cannot attach FinOps policy") {
		t.Fatal("unknown ARNs need a recovery hint, not a bare 'ARN not supported'")
	}
}

func extractQuotedDOC(t *testing.T, cmd string) string {
	t.Helper()
	const prefix = "DOC='"
	start := strings.Index(cmd, prefix)
	if start < 0 {
		t.Fatalf("DOC='...' not found: %s", cmd)
	}
	rest := cmd[start+len(prefix):]
	end := strings.Index(rest, "'")
	if end < 0 {
		t.Fatalf("unclosed DOC quote: %s", cmd)
	}
	return rest[:end]
}
