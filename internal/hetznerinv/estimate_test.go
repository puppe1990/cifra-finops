package hetznerinv

import (
	"testing"

	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func TestEstimate_serverUsesMonthlyCap(t *testing.T) {
	inv := Estimate(Snapshot{
		Pricing: Pricing{
			ServerTypes: []TypePrice{{
				Name: "cx22",
				ByLocation: map[string]LocationPrice{
					"fsn1": {MonthlyCents: 349},
				},
			}},
		},
		Servers: []Server{{
			ID:       42,
			Name:     "web-1",
			Status:   "running",
			Type:     "cx22",
			Location: "fsn1",
		}},
	})

	if inv.Source != "hetzner" {
		t.Fatalf("source = %q", inv.Source)
	}
	if len(inv.Resources) != 1 {
		t.Fatalf("resources = %#v", inv.Resources)
	}
	r := inv.Resources[0]
	if r.Kind != "hetzner_server" || r.Name != "web-1" || r.MonthlyCents != 349 {
		t.Fatalf("resource = %#v", r)
	}
	if len(inv.Lines) != 1 {
		t.Fatalf("lines = %#v", inv.Lines)
	}
	line := inv.Lines[0]
	if line.Service != "Cloud Server" || line.UsageType != "cx22" || line.MonthlyCents != 349 {
		t.Fatalf("line = %#v", line)
	}
}

func TestEstimate_unattachedVolumeIsBilled(t *testing.T) {
	inv := Estimate(Snapshot{
		Pricing: Pricing{VolumePerGB: 0.05},
		Volumes: []Volume{{
			ID: 7, Name: "orphan-disk", Location: "fsn1", SizeGB: 100,
		}},
	})
	if len(inv.Resources) != 1 || inv.Resources[0].MonthlyCents != 500 {
		t.Fatalf("resources = %#v", inv.Resources)
	}
	if !hasService(inv.Lines, "Volume", 500) {
		t.Fatalf("lines = %#v", inv.Lines)
	}
	if !hasFinding(inv.Findings, finops.FindingUnattachedVolume) {
		t.Fatalf("findings = %#v", inv.Findings)
	}
}

func TestEstimate_trafficOverage(t *testing.T) {
	const tib = 1024 * 1024 * 1024 * 1024
	inv := Estimate(Snapshot{
		Pricing: Pricing{
			ServerTypes: []TypePrice{{
				Name: "cx22",
				ByLocation: map[string]LocationPrice{
					"fsn1": {MonthlyCents: 349, IncludedBytes: 20 * tib, TrafficPerTB: 100},
				},
			}},
		},
		Servers: []Server{{
			Name: "web-1", Type: "cx22", Location: "fsn1", Status: "running",
			IncludedTraffic: 20 * tib,
			OutgoingTraffic: 22 * tib,
		}},
	})
	if !hasService(inv.Lines, "Traffic", 200) {
		t.Fatalf("lines = %#v", inv.Lines)
	}
}

func TestEstimate_unattachedPrimaryIP(t *testing.T) {
	inv := Estimate(Snapshot{
		Pricing: Pricing{
			PrimaryIPv4: map[string]LocationPrice{"fsn1": {MonthlyCents: 50}},
		},
		PrimaryIPs: []Address{{
			ID: 9, Name: "spare-ip", Type: "ipv4", Location: "fsn1",
		}},
	})
	if !hasService(inv.Lines, "Primary IP", 50) {
		t.Fatalf("lines = %#v", inv.Lines)
	}
	if !hasFinding(inv.Findings, finops.FindingUnattachedIP) {
		t.Fatalf("findings = %#v", inv.Findings)
	}
}

func TestEstimate_floatingIPLoadBalancerSnapshotBackup(t *testing.T) {
	inv := Estimate(Snapshot{
		Pricing: Pricing{
			ServerTypes: []TypePrice{{
				Name:       "cx22",
				ByLocation: map[string]LocationPrice{"fsn1": {MonthlyCents: 1000}},
			}},
			LoadBalancerTypes: []TypePrice{{
				Name:       "lb11",
				ByLocation: map[string]LocationPrice{"fsn1": {MonthlyCents: 539}},
			}},
			FloatingIPv4:  map[string]LocationPrice{"fsn1": {MonthlyCents: 380}},
			ImagePerGB:    0.01,
			BackupPercent: 20,
		},
		Servers: []Server{{
			ID: 1, Name: "web", Type: "cx22", Location: "fsn1", Status: "running", Backups: true,
		}},
		FloatingIPs: []Address{{
			ID: 2, Name: "float", Type: "ipv4", Location: "fsn1",
		}},
		LoadBalancers: []LoadBalancer{{
			ID: 3, Name: "lb-pub", Type: "lb11", Location: "fsn1",
		}},
		Snapshots: []SnapshotImage{{
			ID: 4, Name: "disk-copy", SizeGB: 10,
		}},
	})
	if !hasService(inv.Lines, "Cloud Server", 1000) {
		t.Fatalf("server missing: %#v", inv.Lines)
	}
	if !hasService(inv.Lines, "Backup", 200) {
		t.Fatalf("backup missing: %#v", inv.Lines)
	}
	if !hasService(inv.Lines, "Floating IP", 380) {
		t.Fatalf("floating missing: %#v", inv.Lines)
	}
	if !hasFinding(inv.Findings, finops.FindingUnattachedIP) {
		t.Fatalf("unattached floating: %#v", inv.Findings)
	}
	if !hasService(inv.Lines, "Load Balancer", 539) {
		t.Fatalf("lb missing: %#v", inv.Lines)
	}
	if !hasService(inv.Lines, "Snapshot", 10) {
		t.Fatalf("snapshot missing: %#v", inv.Lines)
	}
}

func TestEstimate_offServerStillBilled(t *testing.T) {
	inv := Estimate(Snapshot{
		Pricing: Pricing{
			ServerTypes: []TypePrice{{
				Name:       "cx22",
				ByLocation: map[string]LocationPrice{"fsn1": {MonthlyCents: 349}},
			}},
		},
		Servers: []Server{{Name: "idle", Type: "cx22", Location: "fsn1", Status: "off"}},
	})
	if !hasService(inv.Lines, "Cloud Server", 349) {
		t.Fatalf("lines = %#v", inv.Lines)
	}
	if !hasFinding(inv.Findings, finops.FindingStoppedBill) {
		t.Fatalf("findings = %#v", inv.Findings)
	}
}

func hasService(lines []models.CostLine, service string, cents int64) bool {
	for _, l := range lines {
		if l.Service == service && l.MonthlyCents == cents {
			return true
		}
	}
	return false
}

func hasFinding(findings []models.Finding, kind string) bool {
	for _, f := range findings {
		if f.Kind == kind {
			return true
		}
	}
	return false
}
