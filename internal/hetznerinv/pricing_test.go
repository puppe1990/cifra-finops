package hetznerinv

import "testing"

func TestParsePricing_netAmountsToCents(t *testing.T) {
	p, err := ParsePricing([]byte(`{
	  "pricing": {
	    "currency": "EUR",
	    "volume": { "price_per_gb_month": { "net": "0.0440" } },
	    "image": { "price_per_gb_month": { "net": "0.0110" } },
	    "server_backup": { "percentage": "20.00" },
	    "server_types": [{
	      "name": "cx22",
	      "prices": [{
	        "location": "fsn1",
	        "price_monthly": { "net": "3.4900" },
	        "price_hourly": { "net": "0.0052" },
	        "included_traffic": 21990232555520,
	        "price_per_tb_traffic": { "net": "1.0000" }
	      }]
	    }],
	    "primary_ips": [{
	      "type": "ipv4",
	      "prices": [{ "location": "fsn1", "price_monthly": { "net": "0.5000" } }]
	    }],
	    "floating_ips": [{
	      "type": "ipv4",
	      "prices": [{ "location": "fsn1", "price_monthly": { "net": "3.8012" } }]
	    }]
	  }
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if p.VolumePerGB != 0.044 {
		t.Fatalf("volume per gb = %v", p.VolumePerGB)
	}
	if p.ImagePerGB != 0.011 {
		t.Fatalf("image per gb = %v", p.ImagePerGB)
	}
	if p.BackupPercent != 20 {
		t.Fatalf("backup = %v", p.BackupPercent)
	}
	loc, ok := p.lookupServer("cx22", "fsn1")
	if !ok || loc.MonthlyCents != 349 {
		t.Fatalf("cx22 = %#v ok=%v", loc, ok)
	}
	if loc.TrafficPerTB != 100 {
		t.Fatalf("traffic = %d", loc.TrafficPerTB)
	}
	if p.PrimaryIPv4["fsn1"].MonthlyCents != 50 {
		t.Fatalf("primary = %#v", p.PrimaryIPv4)
	}
	if p.FloatingIPv4["fsn1"].MonthlyCents != 380 {
		t.Fatalf("floating = %#v", p.FloatingIPv4)
	}
	if p.Currency != "EUR" {
		t.Fatalf("currency = %q, want EUR", p.Currency)
	}
}

func TestParsePricing_readsUSD(t *testing.T) {
	p, err := ParsePricing([]byte(`{
	  "pricing": {
	    "currency": "USD",
	    "vat_rate": "0.000000",
	    "volume": { "price_per_gb_month": { "net": "0.0000" } },
	    "image": { "price_per_gb_month": { "net": "0.0000" } },
	    "server_backup": { "percentage": "20.00" },
	    "server_types": [{
	      "name": "cx33",
	      "prices": [{
	        "location": "fsn1",
	        "price_monthly": { "net": "9.9900" },
	        "price_hourly": { "net": "0.0160" },
	        "included_traffic": 0,
	        "price_per_tb_traffic": { "net": "1.0000" }
	      }]
	    }]
	  }
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if p.Currency != "USD" {
		t.Fatalf("currency = %q, want USD", p.Currency)
	}
	loc, ok := p.lookupServer("cx33", "fsn1")
	if !ok || loc.MonthlyCents != 999 {
		t.Fatalf("cx33 = %#v ok=%v", loc, ok)
	}
}
