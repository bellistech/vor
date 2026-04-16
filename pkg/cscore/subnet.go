package cscore

import (
	"github.com/bellistech/cs/internal/subnet"
)

type subnetResponse struct {
	CIDR        string `json:"cidr"`
	Network     string `json:"network"`
	Broadcast   string `json:"broadcast,omitempty"`
	Netmask     string `json:"netmask,omitempty"`
	Wildcard    string `json:"wildcard,omitempty"`
	Prefix      int    `json:"prefix"`
	FirstHost   string `json:"first_host"`
	LastHost    string `json:"last_host"`
	TotalHosts  string `json:"total_hosts"`
	UsableHosts string `json:"usable_hosts"`
	IsIPv6      bool   `json:"is_ipv6"`
}

// SubnetCalc parses a CIDR or IP+mask and returns subnet info as JSON.
func SubnetCalc(input string) string {
	if err := validateCIDR(input); err != nil {
		return errorJSON(err)
	}

	info, err := subnet.Calculate(input)
	if err != nil {
		return errorJSON(err)
	}

	resp := subnetResponse{
		CIDR:        info.CIDR,
		Network:     info.Network.String(),
		Prefix:      info.Prefix,
		FirstHost:   info.FirstHost.String(),
		LastHost:    info.LastHost.String(),
		TotalHosts:  info.TotalHosts.String(),
		UsableHosts: info.UsableHosts.String(),
		IsIPv6:      info.IsIPv6,
	}

	if info.Broadcast != nil {
		resp.Broadcast = info.Broadcast.String()
	}
	if info.Netmask != nil {
		resp.Netmask = info.Netmask.String()
	}
	if info.Wildcard != nil {
		resp.Wildcard = info.Wildcard.String()
	}

	return jsonMarshal(resp)
}
