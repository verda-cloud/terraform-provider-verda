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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

func shortenPrivateNetworkDeleteRetry(t *testing.T) {
	t.Helper()
	initial, maxWait := privateNetworkDeleteRetryInitial, privateNetworkDeleteRetryMax
	privateNetworkDeleteRetryInitial, privateNetworkDeleteRetryMax = time.Millisecond, 4*time.Millisecond
	t.Cleanup(func() { privateNetworkDeleteRetryInitial, privateNetworkDeleteRetryMax = initial, maxWait })
}

func TestDeleteRetryingConflicts(t *testing.T) {
	shortenPrivateNetworkDeleteRetry(t)
	conflict := &verda.APIError{StatusCode: 409, Message: "subnet still has 1 attached instance(s): inst_1"}

	t.Run("succeeds first time", func(t *testing.T) {
		calls := 0
		err := deleteRetryingConflicts(context.Background(), func(context.Context) error { calls++; return nil })
		if err != nil || calls != 1 {
			t.Fatalf("err = %v, calls = %d", err, calls)
		}
	})

	t.Run("404 is success", func(t *testing.T) {
		err := deleteRetryingConflicts(context.Background(), func(context.Context) error {
			return &verda.APIError{StatusCode: 404}
		})
		if err != nil {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("retries 409 until success", func(t *testing.T) {
		calls := 0
		err := deleteRetryingConflicts(context.Background(), func(context.Context) error {
			calls++
			if calls < 4 {
				return conflict
			}
			return nil
		})
		if err != nil || calls != 4 {
			t.Fatalf("err = %v, calls = %d", err, calls)
		}
	})

	t.Run("409 then 404 is success", func(t *testing.T) {
		calls := 0
		err := deleteRetryingConflicts(context.Background(), func(context.Context) error {
			calls++
			if calls == 1 {
				return conflict
			}
			return &verda.APIError{StatusCode: 404}
		})
		if err != nil {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("other errors are not retried", func(t *testing.T) {
		calls := 0
		want := &verda.APIError{StatusCode: 500}
		err := deleteRetryingConflicts(context.Background(), func(context.Context) error { calls++; return want })
		if !errors.Is(err, want) || calls != 1 {
			t.Fatalf("err = %v, calls = %d", err, calls)
		}
	})

	t.Run("timeout surfaces the last API message", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		err := deleteRetryingConflicts(ctx, func(context.Context) error { return conflict })
		if err == nil || !strings.Contains(err.Error(), "inst_1") || !isConflict(err) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("cancellation stops retrying", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := deleteRetryingConflicts(ctx, func(context.Context) error { return conflict })
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestFindPrivateNetwork(t *testing.T) {
	networks := []verda.PrivateNetwork{
		{ID: "pn_default", Name: "default", IsDefault: true},
		{ID: "pn_prod", Name: "prod"},
		{ID: "pn_dup1", Name: "dup"},
		{ID: "pn_dup2", Name: "dup"},
	}
	tests := map[string]struct {
		networks []verda.PrivateNetwork
		name     string
		wantID   string
		wantErr  string
	}{
		"by name":       {networks: networks, name: "prod", wantID: "pn_prod"},
		"default":       {networks: networks, wantID: "pn_default"},
		"missing name":  {networks: networks, name: "nope", wantErr: "no private network named"},
		"ambiguous":     {networks: networks, name: "dup", wantErr: "2 private networks"},
		"no default":    {networks: networks[1:], wantErr: "no private network marked"},
		"empty listing": {name: "prod", wantErr: "no private network named"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := findPrivateNetwork(tt.networks, tt.name)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got.ID != tt.wantID {
				t.Fatalf("got %v, %v; want %s", got, err, tt.wantID)
			}
		})
	}
}

func TestFindSubnet(t *testing.T) {
	fin03, fin01 := verda.Location{Code: "FIN-03"}, verda.Location{Code: "FIN-01"}
	subnets := []verda.Subnet{
		{ID: "sn_a3", Name: "app", Location: fin03, IsDefault: true},
		{ID: "sn_b3", Name: "batch", Location: fin03},
		{ID: "sn_a1", Name: "app", Location: fin01},
		{ID: "sn_d1", Name: "db", Location: fin01, IsDefault: true},
	}
	tests := map[string]struct {
		name, location string
		wantID         string
		wantErr        string
	}{
		"name unique across locations": {name: "batch", wantID: "sn_b3"},
		"name narrowed by location":    {name: "app", location: "FIN-01", wantID: "sn_a1"},
		"name ambiguous":               {name: "app", wantErr: "2 subnets"},
		"name in wrong location":       {name: "db", location: "FIN-03", wantErr: "no subnet named \"db\" in FIN-03"},
		"default FIN-03":               {location: "FIN-03", wantID: "sn_a3"},
		"default FIN-01":               {location: "FIN-01", wantID: "sn_d1"},
		"default unknown location":     {location: "FIN-02", wantErr: "no subnet marked as the default in FIN-02"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := findSubnet(subnets, tt.name, tt.location)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got.ID != tt.wantID {
				t.Fatalf("got %v, %v; want %s", got, err, tt.wantID)
			}
		})
	}
}

// TestSchemasValid checks every registered resource and data source schema
// passes the framework's own validation.
func TestSchemasValid(t *testing.T) {
	ctx := context.Background()
	p := &VerdaProvider{}

	for _, newResource := range p.Resources(ctx) {
		r := newResource()
		meta := &resource.MetadataResponse{}
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "verda"}, meta)
		t.Run(meta.TypeName, func(t *testing.T) {
			resp := &resource.SchemaResponse{}
			r.Schema(ctx, resource.SchemaRequest{}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
			}
			if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
				t.Fatalf("invalid schema: %v", diags)
			}
		})
	}
	for _, newDataSource := range p.DataSources(ctx) {
		d := newDataSource()
		meta := &datasource.MetadataResponse{}
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "verda"}, meta)
		t.Run("data."+meta.TypeName, func(t *testing.T) {
			resp := &datasource.SchemaResponse{}
			d.Schema(ctx, datasource.SchemaRequest{}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
			}
			if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
				t.Fatalf("invalid schema: %v", diags)
			}
		})
	}
}

func TestFindRoute(t *testing.T) {
	routes := []verda.Route{
		{ID: "rt_1", Destination: "0.0.0.0/0"},
		{ID: "rt_2", Destination: "192.168.0.0/16"},
	}
	tests := map[string]struct {
		id, destination string
		wantID          string
	}{
		"by id":                      {id: "rt_2", destination: "192.168.0.0/16", wantID: "rt_2"},
		"id wins over destination":   {id: "rt_2", destination: "0.0.0.0/0", wantID: "rt_2"},
		"id gone, destination found": {id: "rt_old", destination: "0.0.0.0/0", wantID: "rt_1"},
		"neither matches":            {id: "rt_old", destination: "10.0.0.0/8"},
		"import before first read":   {id: "rt_old"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := findRoute(routes, tt.id, tt.destination)
			if tt.wantID == "" {
				if got != nil {
					t.Fatalf("got %v, want nil", got)
				}
				return
			}
			if got == nil || got.ID != tt.wantID {
				t.Fatalf("got %v, want %s", got, tt.wantID)
			}
		})
	}
}

func TestSetNetworkRouteModel(t *testing.T) {
	data := NetworkRouteResourceModel{}
	setNetworkRouteModel(&data, &verda.Route{ID: "rt_1", SubnetID: "sn_1", Destination: "0.0.0.0/0", Instance: &verda.AttachedInstance{ID: "inst_1"}})
	if data.ID.ValueString() != "rt_1" || data.SubnetID.ValueString() != "sn_1" || data.InstanceID.ValueString() != "inst_1" {
		t.Fatalf("got %+v", data)
	}
	// A route whose instance is gone has no instance_id, so the plan shows
	// the configured one being set again.
	setNetworkRouteModel(&data, &verda.Route{ID: "rt_1", Destination: "0.0.0.0/0"})
	if !data.InstanceID.IsNull() || data.SubnetID.ValueString() != "sn_1" {
		t.Fatalf("got %+v", data)
	}
}
