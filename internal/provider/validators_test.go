// Copyright 2026 Verda Cloud Oy
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCheckSubnetCIDR(t *testing.T) {
	valid := []string{
		"10.0.0.0/8",
		"10.10.0.0/24",
		"10.255.255.248/29", // last /29 in 10/8
		"172.16.0.0/12",
		"172.31.255.0/24", // top of 172.16/12
		"192.168.0.0/16",
		"192.168.1.0/29",
	}
	invalid := map[string]string{
		"garbage":                "not-a-cidr",
		"bare address":           "10.0.0.1",
		"IPv6":                   "fd00::/64",
		"host bits set":          "10.0.0.5/24",
		"/30 too small":          "10.0.0.0/30",
		"/32 too small":          "10.0.0.1/32",
		"public range":           "8.8.8.0/24",
		"wider than 10/8":        "10.0.0.0/7",
		"wider than 172.16/12":   "172.16.0.0/11",
		"just below 172.16/12":   "172.15.255.0/24",
		"just above 172.16/12":   "172.32.0.0/24",
		"wider than 192.168/16":  "192.168.0.0/15",
		"just above 192.168/16":  "192.169.0.0/24",
		"CGNAT not RFC 1918":     "100.64.0.0/24",
		"spans 10/8 boundary":    "8.0.0.0/6",
		"default route":          "0.0.0.0/0",
		"IPv4-mapped IPv6":       "::ffff:10.0.0.0/120",
		"surrounding whitespace": " 10.0.0.0/24",
	}

	for _, s := range valid {
		t.Run("valid "+s, func(t *testing.T) {
			if err := checkSubnetCIDR(s); err != nil {
				t.Errorf("checkSubnetCIDR(%q) = %v, want nil", s, err)
			}
		})
	}
	for name, s := range invalid {
		t.Run("invalid "+name, func(t *testing.T) {
			if err := checkSubnetCIDR(s); err == nil {
				t.Errorf("checkSubnetCIDR(%q) = nil, want error", s)
			}
		})
	}
}

func TestCheckSubnetCIDRSuggestsNetworkAddress(t *testing.T) {
	err := checkSubnetCIDR("10.0.0.5/24")
	if err == nil || !strings.Contains(err.Error(), `"10.0.0.0/24"`) {
		t.Errorf("expected suggestion of 10.0.0.0/24, got %v", err)
	}
}

// runStringValidator runs v against value and reports whether it produced an error.
func runStringValidator(v validator.String, value types.String) bool {
	req := validator.StringRequest{Path: path.Root("test"), ConfigValue: value}
	resp := &validator.StringResponse{}
	v.ValidateString(context.Background(), req, resp)
	return resp.Diagnostics.HasError()
}

func TestStringValidators(t *testing.T) {
	tests := []struct {
		name    string
		v       validator.String
		value   types.String
		wantErr bool
	}{
		{"cidr null", subnetCIDRValidator{}, types.StringNull(), false},
		{"cidr unknown", subnetCIDRValidator{}, types.StringUnknown(), false},
		{"cidr ok", subnetCIDRValidator{}, types.StringValue("10.10.0.0/24"), false},
		{"cidr bad", subnetCIDRValidator{}, types.StringValue("10.10.0.0/30"), true},

		{"private null", privateIPValidator{}, types.StringNull(), false},
		{"private unknown", privateIPValidator{}, types.StringUnknown(), false},
		{"private auto", privateIPValidator{}, types.StringValue("auto"), false},
		{"private ipv4", privateIPValidator{}, types.StringValue("10.10.0.5"), false},
		{"private none", privateIPValidator{}, types.StringValue("none"), true},
		{"private cidr", privateIPValidator{}, types.StringValue("10.10.0.5/24"), true},
		{"private ipv6", privateIPValidator{}, types.StringValue("fd00::1"), true},
		{"private AUTO", privateIPValidator{}, types.StringValue("AUTO"), true},
		{"private empty", privateIPValidator{}, types.StringValue(""), true},

		{"public null", publicIPValidator{}, types.StringNull(), false},
		{"public unknown", publicIPValidator{}, types.StringUnknown(), false},
		{"public auto", publicIPValidator{}, types.StringValue("auto"), false},
		{"public none", publicIPValidator{}, types.StringValue("none"), false},
		{"public literal", publicIPValidator{}, types.StringValue("1.2.3.4"), true},
		{"public empty", publicIPValidator{}, types.StringValue(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runStringValidator(tt.v, tt.value); got != tt.wantErr {
				t.Errorf("error = %v, want %v", got, tt.wantErr)
			}
		})
	}
}
