package finops

import "regexp"

const (
	// AmplifyService is the Cost Explorer SERVICE value for AWS Amplify.
	AmplifyService = "AWS Amplify"
	// NoUsageType is the dimension used when a cost line has no USAGE_TYPE.
	NoUsageType = "NoUsageType"
)

// amplifyRegionPrefix matches the region code CE prepends to some Amplify usage
// types, e.g. USE1-BuildDuration, EU-DataTransferOut, APS3-BuildDuration.
// Only a single leading region prefix is stripped.
var amplifyRegionPrefix = regexp.MustCompile(`^[A-Z]{2,4}[0-9]*-`)

// AmplifyDimension collapses region variants into one billing dimension:
// USE1-BuildDuration -> BuildDuration.
func AmplifyDimension(usageType string) string {
	dimension := amplifyRegionPrefix.ReplaceAllString(usageType, "")
	if dimension == "" {
		return NoUsageType
	}
	return dimension
}
