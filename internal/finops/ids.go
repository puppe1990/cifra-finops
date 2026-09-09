package finops

import (
	"os"
	"strings"
)

const (
	SeedAccountEnv = "CIFRA_SEED_AWS_ACCOUNT_ID"

	PrimaryTenantSlug   = "demo"
	PrimaryTenantName   = "Demo"
	PrimaryAccountAlias = "principal"
	DefaultRegion       = "us-east-1"

	AuthModeDefaultChain = "default_chain"
	AuthModeAccessKeys   = "access_keys"

	RoleOwner  = "owner"
	RoleMember = "member"

	SourceEstimate = "estimate"
	SourceCE       = "ce"
	SourceHetzner  = "hetzner"

	ProviderAWS     = "aws"
	ProviderHetzner = "hetzner"

	AuthModeAPIToken = "api_token"

	SyncOK     = "ok"
	SyncFailed = "failed"

	FindingCEDenied         = "ce_denied"
	FindingUnattachedIP     = "unattached_ip"
	FindingUnknownS3Size    = "unknown_s3_size"
	FindingStoppedBill      = "stopped_instance_billed"
	FindingUnattachedVolume = "unattached_volume"

	DefaultHetznerRegion = "fsn1"
)

func SeedAWSAccountID() string {
	return strings.TrimSpace(os.Getenv(SeedAccountEnv))
}

func ValidAWSAccountID(id string) bool {
	if len(id) != 12 {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func HetznerProjectID(alias string) string {
	slug := strings.ToLower(strings.TrimSpace(alias))
	slug = strings.ReplaceAll(slug, " ", "-")
	return "hz:" + slug
}

func ValidHetznerToken(token string) bool {
	return len(strings.TrimSpace(token)) >= 16
}
