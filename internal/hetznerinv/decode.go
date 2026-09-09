package hetznerinv

type pageMeta struct {
	Meta struct {
		Pagination struct {
			NextPage *int `json:"next_page"`
		} `json:"pagination"`
	} `json:"meta"`
}

type serverPage struct {
	pageMeta
	Servers []apiServer `json:"servers"`
}

type apiServer struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	ServerType struct {
		Name string `json:"name"`
	} `json:"server_type"`
	Location   named `json:"location"`
	Datacenter struct {
		Location named `json:"location"`
	} `json:"datacenter"`
	PublicNet struct {
		IPv4 *struct {
			ID int64 `json:"id"`
		} `json:"ipv4"`
	} `json:"public_net"`
	IncludedTraffic uint64  `json:"included_traffic"`
	OutgoingTraffic *uint64 `json:"outgoing_traffic"`
	BackupWindow    *string `json:"backup_window"`
}

func (s apiServer) locationName() string {
	if s.Location.Name != "" {
		return s.Location.Name
	}
	return s.Datacenter.Location.Name
}

type named struct {
	Name string `json:"name"`
}

type volumePage struct {
	pageMeta
	Volumes []apiVolume `json:"volumes"`
}

type apiVolume struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Size     int    `json:"size"`
	Server   *int64 `json:"server"`
	Location named  `json:"location"`
}

type addressPage struct {
	pageMeta
	PrimaryIPs  []apiAddress `json:"primary_ips"`
	FloatingIPs []apiAddress `json:"floating_ips"`
}

type apiAddress struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	AssigneeID   *int64 `json:"assignee_id"`
	Server       *int64 `json:"server"`
	Datacenter   *named `json:"datacenter"`
	HomeLocation *named `json:"home_location"`
	Location     *named `json:"location"`
}

func (a apiAddress) locationName() string {
	for _, n := range []*named{a.Location, a.HomeLocation, a.Datacenter} {
		if n != nil && n.Name != "" {
			return n.Name
		}
	}
	return ""
}

func (a apiAddress) assigned() bool {
	return a.AssigneeID != nil || a.Server != nil
}

type loadBalancerPage struct {
	pageMeta
	LoadBalancers []apiLoadBalancer `json:"load_balancers"`
}

type apiLoadBalancer struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	LoadBalancerType struct {
		Name string `json:"name"`
	} `json:"load_balancer_type"`
	Location named `json:"location"`
}

type imagePage struct {
	pageMeta
	Images []apiImage `json:"images"`
}

type apiImage struct {
	ID          int64    `json:"id"`
	Description string   `json:"description"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	ImageSize   *float64 `json:"image_size"`
}

func (img apiImage) label() string {
	if img.Description != "" {
		return img.Description
	}
	if img.Name != "" {
		return img.Name
	}
	return "snapshot"
}
