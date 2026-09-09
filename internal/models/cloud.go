package models

import "time"

type CloudAccount struct {
	ID           int64
	TenantID     int64
	Provider     string
	AWSAccountID string
	Alias        string
	Region       string
	AuthMode     string
	AccessKeyID  string
	SecretCipher string
	IsPrimary    bool
	CreatedAt    time.Time
}

type CloudResource struct {
	ID             int64
	CloudAccountID int64
	Kind           string
	Name           string
	Region         string
	State          string
	MonthlyCents   int64
	Source         string
	ExternalID     string
	MetaJSON       string
}

type CostLine struct {
	ID             int64
	CloudAccountID int64
	Service        string
	UsageType      string
	MonthlyCents   int64
	Source         string
	PeriodStart    string
	PeriodEnd      string
}

type Finding struct {
	ID             int64
	CloudAccountID int64
	Kind           string
	Severity       string
	Title          string
	Detail         string
}

type Budget struct {
	ID             int64
	TenantID       int64
	CloudAccountID int64
	Name           string
	AmountCents    int64
	Period         string
}

type SyncRun struct {
	ID             int64
	CloudAccountID int64
	Status         string
	Source         string
	Error          string
	Warning        string
	StartedAt      time.Time
	FinishedAt     time.Time
}
