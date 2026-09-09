package awsinv

import (
	"bytes"
	"encoding/json"
	"strings"
)

const FinOpsPolicyName = "CifraFinOpsRead"
const FinOpsIAMUser = "cifra-finops"

const FinOpsIAMPolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "CostExplorerAndBudgets",
      "Effect": "Allow",
      "Action": [
        "ce:GetCostAndUsage",
        "ce:GetCostForecast",
        "ce:GetRightsizingRecommendation",
        "ce:GetAnomalies",
        "budgets:ViewBudget"
      ],
      "Resource": "*"
    },
    {
      "Sid": "InventoryRead",
      "Effect": "Allow",
      "Action": [
        "sts:GetCallerIdentity",
        "s3:ListAllMyBuckets",
        "s3:GetBucketLocation",
        "s3:ListBucket",
        "lightsail:GetInstances",
        "lightsail:GetBundles",
        "lightsail:GetStaticIps",
        "lightsail:GetDisks",
        "lightsail:GetInstanceSnapshots",
        "lightsail:GetRelationalDatabases",
        "cloudwatch:GetMetricStatistics"
      ],
      "Resource": "*"
    }
  ]
}`

func CompactIAMPolicy() string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(FinOpsIAMPolicy)); err != nil {
		return FinOpsIAMPolicy
	}
	return buf.String()
}

func CloudShellCommand() string {
	cmd := `DOC='{{DOC}}'; ARN=$(aws sts get-caller-identity --query Arn --output text); case "$ARN" in *:user/*) aws iam put-user-policy --user-name "${ARN##*/}" --policy-name {{POLICY}} --policy-document "$DOC" && echo "OK: {{POLICY}} applied to IAM user ${ARN##*/}";; *:assumed-role/*) aws iam put-role-policy --role-name "$(echo "$ARN"|cut -d/ -f2)" --policy-name {{POLICY}} --policy-document "$DOC" && echo "OK: {{POLICY}} applied to role $(echo "$ARN"|cut -d/ -f2)";; *:root) aws iam create-user --user-name {{USER}} >/dev/null 2>&1 || true; aws iam put-user-policy --user-name {{USER}} --policy-name {{POLICY}} --policy-document "$DOC"; echo "CloudShell is account root — IAM policies cannot be attached to root. Applied {{POLICY}} to IAM user {{USER}}. Paste this user's access keys in Cifra → Accounts (never use root keys)."; aws iam create-access-key --user-name {{USER}} || { echo "Cannot create another access key for {{USER}} — AWS quota is 2 per user and the secret of existing keys cannot be retrieved. Delete an unused key, then create:"; aws iam list-access-keys --user-name {{USER}}; echo "aws iam delete-access-key --user-name {{USER}} --access-key-id KEYID && aws iam create-access-key --user-name {{USER}}"; };; *) echo "Cannot attach FinOps policy to $ARN. Sign in as an IAM user or role, or as account root (creates IAM user {{USER}})."; false;; esac`
	cmd = strings.ReplaceAll(cmd, "{{DOC}}", CompactIAMPolicy())
	cmd = strings.ReplaceAll(cmd, "{{POLICY}}", FinOpsPolicyName)
	return strings.ReplaceAll(cmd, "{{USER}}", FinOpsIAMUser)
}
