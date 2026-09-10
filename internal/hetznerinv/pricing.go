package hetznerinv

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type pricingPayload struct {
	Pricing struct {
		Currency string `json:"currency"`
		Volume   struct {
			PricePerGBMonth priceAmount `json:"price_per_gb_month"`
		} `json:"volume"`
		Image struct {
			PricePerGBMonth priceAmount `json:"price_per_gb_month"`
		} `json:"image"`
		ServerBackup struct {
			Percentage string `json:"percentage"`
		} `json:"server_backup"`
		ServerTypes []struct {
			Name   string             `json:"name"`
			Prices []typedLocationAmt `json:"prices"`
		} `json:"server_types"`
		LoadBalancerTypes []struct {
			Name   string             `json:"name"`
			Prices []typedLocationAmt `json:"prices"`
		} `json:"load_balancer_types"`
		PrimaryIPs []struct {
			Type   string             `json:"type"`
			Prices []namedLocationAmt `json:"prices"`
		} `json:"primary_ips"`
		FloatingIPs []struct {
			Type   string             `json:"type"`
			Prices []namedLocationAmt `json:"prices"`
		} `json:"floating_ips"`
	} `json:"pricing"`
}

type priceAmount struct {
	Net string `json:"net"`
}

type typedLocationAmt struct {
	Location          string      `json:"location"`
	PriceMonthly      priceAmount `json:"price_monthly"`
	PriceHourly       priceAmount `json:"price_hourly"`
	IncludedTraffic   uint64      `json:"included_traffic"`
	PricePerTBTraffic priceAmount `json:"price_per_tb_traffic"`
}

type namedLocationAmt struct {
	Location     string      `json:"location"`
	PriceMonthly priceAmount `json:"price_monthly"`
}

func ParsePricing(raw []byte) (Pricing, error) {
	var payload pricingPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Pricing{}, fmt.Errorf("hetzner pricing: %w", err)
	}
	src := payload.Pricing
	out := Pricing{
		Currency:      strings.ToUpper(strings.TrimSpace(src.Currency)),
		VolumePerGB:   parseEuros(src.Volume.PricePerGBMonth.Net),
		ImagePerGB:    parseEuros(src.Image.PricePerGBMonth.Net),
		BackupPercent: parseEuros(src.ServerBackup.Percentage),
		PrimaryIPv4:   map[string]LocationPrice{},
		FloatingIPv4:  map[string]LocationPrice{},
	}
	for _, st := range src.ServerTypes {
		out.ServerTypes = append(out.ServerTypes, TypePrice{Name: st.Name, ByLocation: locationPrices(st.Prices)})
	}
	for _, lb := range src.LoadBalancerTypes {
		out.LoadBalancerTypes = append(out.LoadBalancerTypes, TypePrice{Name: lb.Name, ByLocation: locationPrices(lb.Prices)})
	}
	for _, ip := range src.PrimaryIPs {
		if ip.Type != "ipv4" {
			continue
		}
		for _, loc := range ip.Prices {
			out.PrimaryIPv4[loc.Location] = LocationPrice{MonthlyCents: eurosToCents(parseEuros(loc.PriceMonthly.Net))}
		}
	}
	for _, ip := range src.FloatingIPs {
		if ip.Type != "ipv4" {
			continue
		}
		for _, loc := range ip.Prices {
			out.FloatingIPv4[loc.Location] = LocationPrice{MonthlyCents: eurosToCents(parseEuros(loc.PriceMonthly.Net))}
		}
	}
	return out, nil
}

func locationPrices(rows []typedLocationAmt) map[string]LocationPrice {
	out := map[string]LocationPrice{}
	for _, row := range rows {
		out[row.Location] = LocationPrice{
			HourlyCents:   eurosToCents(parseEuros(row.PriceHourly.Net)),
			MonthlyCents:  eurosToCents(parseEuros(row.PriceMonthly.Net)),
			IncludedBytes: row.IncludedTraffic,
			TrafficPerTB:  eurosToCents(parseEuros(row.PricePerTBTraffic.Net)),
		}
	}
	return out
}

func parseEuros(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}
