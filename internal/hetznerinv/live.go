package hetznerinv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.hetzner.cloud/v1"

type Live struct {
	BaseURL string
	HTTP    *http.Client
}

func NewLive() *Live {
	return &Live{}
}

func (l *Live) Collect(ctx context.Context, token string) (Inventory, error) {
	if strings.TrimSpace(token) == "" {
		return Inventory{}, fmt.Errorf("hetzner api token is empty")
	}
	pricingRaw, err := l.get(ctx, token, "/pricing", nil)
	if err != nil {
		return Inventory{}, err
	}
	pricing, err := ParsePricing(pricingRaw)
	if err != nil {
		return Inventory{}, err
	}
	snap := Snapshot{Pricing: pricing}
	if err := l.loadInventory(ctx, token, &snap); err != nil {
		return Inventory{}, err
	}
	return Estimate(snap), nil
}

func (l *Live) loadInventory(ctx context.Context, token string, snap *Snapshot) error {
	var servers serverPage
	if err := l.list(ctx, token, "/servers", nil, &servers); err != nil {
		return err
	}
	for _, s := range servers.Servers {
		out := Server{
			ID: s.ID, Name: s.Name, Status: s.Status, Type: s.ServerType.Name,
			Location: s.locationName(), IncludedTraffic: s.IncludedTraffic,
			Backups: s.BackupWindow != nil, PrimaryIPv4: s.PublicNet.IPv4 != nil,
		}
		if s.OutgoingTraffic != nil {
			out.OutgoingTraffic = *s.OutgoingTraffic
		}
		snap.Servers = append(snap.Servers, out)
	}

	var volumes volumePage
	if err := l.list(ctx, token, "/volumes", nil, &volumes); err != nil {
		return err
	}
	for _, v := range volumes.Volumes {
		vol := Volume{ID: v.ID, Name: v.Name, Location: v.Location.Name, SizeGB: v.Size}
		if v.Server != nil {
			vol.ServerID = *v.Server
		}
		snap.Volumes = append(snap.Volumes, vol)
	}

	var primary addressPage
	if err := l.list(ctx, token, "/primary_ips", nil, &primary); err != nil {
		return err
	}
	for _, ip := range primary.PrimaryIPs {
		snap.PrimaryIPs = append(snap.PrimaryIPs, Address{
			ID: ip.ID, Name: ip.Name, Type: ip.Type, Location: ip.locationName(), Assigned: ip.assigned(),
		})
	}

	var floating addressPage
	if err := l.list(ctx, token, "/floating_ips", nil, &floating); err != nil {
		return err
	}
	for _, ip := range floating.FloatingIPs {
		snap.FloatingIPs = append(snap.FloatingIPs, Address{
			ID: ip.ID, Name: ip.Name, Type: ip.Type, Location: ip.locationName(), Assigned: ip.assigned(),
		})
	}

	var lbs loadBalancerPage
	if err := l.list(ctx, token, "/load_balancers", nil, &lbs); err != nil {
		return err
	}
	for _, lb := range lbs.LoadBalancers {
		snap.LoadBalancers = append(snap.LoadBalancers, LoadBalancer{
			ID: lb.ID, Name: lb.Name, Type: lb.LoadBalancerType.Name, Location: lb.Location.Name,
		})
	}

	var images imagePage
	if err := l.list(ctx, token, "/images", url.Values{"type": {"snapshot"}}, &images); err != nil {
		return err
	}
	for _, img := range images.Images {
		size := 0.0
		if img.ImageSize != nil {
			size = *img.ImageSize
		}
		snap.Snapshots = append(snap.Snapshots, SnapshotImage{ID: img.ID, Name: img.label(), SizeGB: size})
	}
	return nil
}

func (l *Live) list(ctx context.Context, token, path string, query url.Values, dest any) error {
	raw, err := l.get(ctx, token, path, query)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("hetzner %s: %w", path, err)
	}
	return nil
}

func (l *Live) get(ctx context.Context, token, path string, query url.Values) ([]byte, error) {
	base := l.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	u, err := url.Parse(strings.TrimRight(base, "/") + path)
	if err != nil {
		return nil, fmt.Errorf("hetzner url: %w", err)
	}
	if query == nil {
		query = url.Values{}
	}
	if query.Get("per_page") == "" {
		query.Set("per_page", "50")
	}
	u.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	client := l.HTTP
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hetzner %s: %w", path, err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("hetzner %s: %w", path, err)
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("hetzner %s: HTTP %d: %s", path, res.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}
