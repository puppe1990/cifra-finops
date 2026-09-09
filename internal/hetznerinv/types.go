package hetznerinv

import "github.com/puppe1990/cifra-finops/internal/models"

type Snapshot struct {
	Pricing       Pricing
	Servers       []Server
	Volumes       []Volume
	PrimaryIPs    []Address
	FloatingIPs   []Address
	LoadBalancers []LoadBalancer
	Snapshots     []SnapshotImage
}

type Pricing struct {
	ServerTypes       []TypePrice
	LoadBalancerTypes []TypePrice
	VolumePerGB       float64
	ImagePerGB        float64
	PrimaryIPv4       map[string]LocationPrice
	FloatingIPv4      map[string]LocationPrice
	BackupPercent     float64
	TrafficPerTB      map[string]int64
}

type TypePrice struct {
	Name       string
	ByLocation map[string]LocationPrice
}

type LocationPrice struct {
	HourlyCents   int64
	MonthlyCents  int64
	IncludedBytes uint64
	TrafficPerTB  int64
}

type Server struct {
	ID              int64
	Name            string
	Status          string
	Type            string
	Location        string
	IncludedTraffic uint64
	OutgoingTraffic uint64
	Backups         bool
	PrimaryIPv4     bool
}

type Volume struct {
	ID       int64
	Name     string
	Location string
	SizeGB   int
	ServerID int64
}

type Address struct {
	ID       int64
	Name     string
	Type     string
	Location string
	Assigned bool
}

type LoadBalancer struct {
	ID       int64
	Name     string
	Type     string
	Location string
}

type SnapshotImage struct {
	ID     int64
	Name   string
	SizeGB float64
}

type Inventory struct {
	Source    string
	Resources []models.CloudResource
	Lines     []models.CostLine
	Findings  []models.Finding
	Warnings  []string
}
