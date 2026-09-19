package finops

import "testing"

func TestAmplifyDimension_stripsRegionPrefix(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "strips us-east-1 prefix", in: "USE1-BuildDuration", want: "BuildDuration"},
		{name: "strips eu prefix", in: "EU-DataTransferOut", want: "DataTransferOut"},
		{name: "strips ap-southeast-3 prefix", in: "APS3-BuildDuration", want: "BuildDuration"},
		{name: "strips ap-northeast-1 prefix", in: "APN1-DataStorage", want: "DataStorage"},
		{name: "keeps unprefixed usage type", in: "HostingComputeRequestCount", want: "HostingComputeRequestCount"},
		{name: "strips prefix before long usage type", in: "USE1-HostingComputeRequestDuration", want: "HostingComputeRequestDuration"},
		{name: "empty usage type", in: "", want: "NoUsageType"},
		{name: "prefix with no remainder", in: "USE1-", want: "NoUsageType"},
		{name: "strips only one leading prefix", in: "USE1-USE1-BuildDuration", want: "USE1-BuildDuration"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := AmplifyDimension(tc.in); got != tc.want {
				t.Errorf("AmplifyDimension(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
