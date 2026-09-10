package hetznerinv

import (
	"math"
	"strconv"
	"time"

	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

const tiB = 1024 * 1024 * 1024 * 1024

func Estimate(snap Snapshot) Inventory {
	start, end := monthBounds(time.Now().UTC())
	inv := Inventory{Source: finops.SourceHetzner, Currency: snap.Pricing.Currency}

	for _, srv := range snap.Servers {
		cents := snap.Pricing.serverMonthly(srv.Type, srv.Location)
		inv.Resources = append(inv.Resources, resource("hetzner_server", srv.Name, srv.Location, srv.Status, cents, srv.ID))
		inv.Lines = append(inv.Lines, costLine("Cloud Server", srv.Type, cents, start, end))
		if srv.Status == "off" {
			inv.Findings = append(inv.Findings, models.Finding{
				Kind:     finops.FindingStoppedBill,
				Severity: "info",
				Title:    srv.Name,
			})
		}
		if over := trafficOverageCents(srv, snap.Pricing); over > 0 {
			inv.Lines = append(inv.Lines, costLine("Traffic", srv.Name, over, start, end))
		}
		if srv.Backups && snap.Pricing.BackupPercent > 0 {
			backup := int64(math.Round(float64(cents) * snap.Pricing.BackupPercent / 100))
			inv.Lines = append(inv.Lines, costLine("Backup", srv.Name, backup, start, end))
		}
	}

	for _, vol := range snap.Volumes {
		cents := eurosToCents(float64(vol.SizeGB) * snap.Pricing.VolumePerGB)
		state := "attached"
		if vol.ServerID == 0 {
			state = "available"
			inv.Findings = append(inv.Findings, models.Finding{
				Kind:     finops.FindingUnattachedVolume,
				Severity: "warning",
				Title:    vol.Name,
			})
		}
		inv.Resources = append(inv.Resources, resource("hetzner_volume", vol.Name, vol.Location, state, cents, vol.ID))
		inv.Lines = append(inv.Lines, costLine("Volume", vol.Name, cents, start, end))
	}

	for _, ip := range snap.PrimaryIPs {
		cents := int64(0)
		if ip.Type == "ipv4" {
			cents = snap.Pricing.PrimaryIPv4[ip.Location].MonthlyCents
		}
		state := "assigned"
		if !ip.Assigned {
			state = "unassigned"
			if cents > 0 {
				inv.Findings = append(inv.Findings, models.Finding{
					Kind:     finops.FindingUnattachedIP,
					Severity: "warning",
					Title:    ip.Name,
				})
			}
		}
		inv.Resources = append(inv.Resources, resource("hetzner_primary_ip", ip.Name, ip.Location, state, cents, ip.ID))
		if cents > 0 {
			inv.Lines = append(inv.Lines, costLine("Primary IP", ip.Type, cents, start, end))
		}
	}

	for _, ip := range snap.FloatingIPs {
		price := snap.Pricing.FloatingIPv4[ip.Location]
		cents := price.MonthlyCents
		state := "assigned"
		if !ip.Assigned {
			state = "unassigned"
			inv.Findings = append(inv.Findings, models.Finding{
				Kind:     finops.FindingUnattachedIP,
				Severity: "warning",
				Title:    ip.Name,
			})
		}
		inv.Resources = append(inv.Resources, resource("hetzner_floating_ip", ip.Name, ip.Location, state, cents, ip.ID))
		inv.Lines = append(inv.Lines, costLine("Floating IP", ip.Type, cents, start, end))
	}

	for _, lb := range snap.LoadBalancers {
		cents := snap.Pricing.loadBalancerMonthly(lb.Type, lb.Location)
		inv.Resources = append(inv.Resources, resource("hetzner_load_balancer", lb.Name, lb.Location, "running", cents, lb.ID))
		inv.Lines = append(inv.Lines, costLine("Load Balancer", lb.Type, cents, start, end))
	}

	for _, img := range snap.Snapshots {
		cents := eurosToCents(img.SizeGB * snap.Pricing.ImagePerGB)
		inv.Resources = append(inv.Resources, resource("hetzner_snapshot", img.Name, "", "available", cents, img.ID))
		inv.Lines = append(inv.Lines, costLine("Snapshot", img.Name, cents, start, end))
	}

	return stampCurrency(inv)
}

func stampCurrency(inv Inventory) Inventory {
	for i := range inv.Resources {
		inv.Resources[i].Currency = inv.Currency
	}
	for i := range inv.Lines {
		inv.Lines[i].Currency = inv.Currency
	}
	return inv
}

func trafficOverageCents(srv Server, p Pricing) int64 {
	included := srv.IncludedTraffic
	perTB := int64(0)
	if loc, ok := p.lookupServer(srv.Type, srv.Location); ok {
		if included == 0 {
			included = loc.IncludedBytes
		}
		perTB = loc.TrafficPerTB
	}
	if srv.OutgoingTraffic <= included || perTB == 0 {
		return 0
	}
	tb := float64(srv.OutgoingTraffic-included) / float64(tiB)
	return int64(math.Round(tb * float64(perTB)))
}

func resource(kind, name, region, state string, cents, id int64) models.CloudResource {
	return models.CloudResource{
		Kind:         kind,
		Name:         name,
		Region:       region,
		State:        state,
		MonthlyCents: cents,
		Source:       finops.SourceHetzner,
		ExternalID:   strconv.FormatInt(id, 10),
	}
}

func costLine(service, usage string, cents int64, start, end string) models.CostLine {
	return models.CostLine{
		Service:      service,
		UsageType:    usage,
		MonthlyCents: cents,
		Source:       finops.SourceHetzner,
		PeriodStart:  start,
		PeriodEnd:    end,
	}
}

func (p Pricing) serverMonthly(typeName, location string) int64 {
	if loc, ok := p.lookupServer(typeName, location); ok {
		return loc.MonthlyCents
	}
	return 0
}

func (p Pricing) loadBalancerMonthly(typeName, location string) int64 {
	for _, tp := range p.LoadBalancerTypes {
		if tp.Name != typeName {
			continue
		}
		if loc, ok := tp.ByLocation[location]; ok {
			return loc.MonthlyCents
		}
	}
	return 0
}

func (p Pricing) lookupServer(typeName, location string) (LocationPrice, bool) {
	for _, tp := range p.ServerTypes {
		if tp.Name != typeName {
			continue
		}
		loc, ok := tp.ByLocation[location]
		return loc, ok
	}
	return LocationPrice{}, false
}

func eurosToCents(euros float64) int64 {
	return int64(math.Round(euros * 100))
}

func monthBounds(now time.Time) (start, end string) {
	s := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	e := s.AddDate(0, 1, 0)
	return s.Format("2006-01-02"), e.Format("2006-01-02")
}
