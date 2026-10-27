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
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

func TestFlattenManagedKubernetesClusterToModel(t *testing.T) {
	r := &ManagedKubernetesClusterResource{}

	cluster := &verda.ManagedKubernetesCluster{
		Name:              "my-cluster",
		KubernetesVersion: "1.32",
		Location:          "FIN-03",
		NetworkID:         "net-1",
		SubnetID:          "sub-1",
		PodCIDR:           "10.244.0.0/16",
		ServiceCIDR:       "10.96.0.0/16",
		DNSSuffix:         "my-cluster.k8s.example.com",
		Phase:             "running",
		ExternalEndpoint:  "https://api.example.com",
		CreatedAt:         time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	var data ManagedKubernetesClusterResourceModel
	r.flattenManagedKubernetesClusterToModel(cluster, &data)

	want := map[string]string{
		"id":                 data.ID.ValueString(),
		"name":               data.Name.ValueString(),
		"kubernetes_version": data.KubernetesVersion.ValueString(),
		"location":           data.Location.ValueString(),
		"network_id":         data.NetworkID.ValueString(),
		"subnet_id":          data.SubnetID.ValueString(),
		"pod_cidr":           data.PodCIDR.ValueString(),
		"service_cidr":       data.ServiceCIDR.ValueString(),
		"dns_suffix":         data.DNSSuffix.ValueString(),
		"phase":              data.Phase.ValueString(),
		"external_endpoint":  data.ExternalEndpoint.ValueString(),
		"created_at":         data.CreatedAt.ValueString(),
	}
	wantValues := map[string]string{
		"id":                 "FIN-03/my-cluster",
		"name":               "my-cluster",
		"kubernetes_version": "1.32",
		"location":           "FIN-03",
		"network_id":         "net-1",
		"subnet_id":          "sub-1",
		"pod_cidr":           "10.244.0.0/16",
		"service_cidr":       "10.96.0.0/16",
		"dns_suffix":         "my-cluster.k8s.example.com",
		"phase":              "running",
		"external_endpoint":  "https://api.example.com",
		"created_at":         "2026-01-02T03:04:05Z",
	}
	for field, got := range want {
		if got != wantValues[field] {
			t.Errorf("%s = %q, want %q", field, got, wantValues[field])
		}
	}
}

// managedKubernetesClusterConfig builds a tfsdk.Config for the cluster schema
// with all attributes null except pod_cidr / service_cidr.
func managedKubernetesClusterConfig(t *testing.T, ctx context.Context, podCIDR, serviceCIDR string) tfsdk.Config {
	t.Helper()

	var schemaResp resource.SchemaResponse
	(&ManagedKubernetesClusterResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics)
	}
	s := schemaResp.Schema

	terraformType := s.Type().TerraformType(ctx)
	objectType, ok := terraformType.(tftypes.Object)
	if !ok {
		t.Fatalf("cluster schema type is %T, want tftypes.Object", terraformType)
	}

	values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attrType := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(attrType, nil)
	}
	if podCIDR != "" {
		values["pod_cidr"] = tftypes.NewValue(tftypes.String, podCIDR)
	}
	if serviceCIDR != "" {
		values["service_cidr"] = tftypes.NewValue(tftypes.String, serviceCIDR)
	}

	return tfsdk.Config{
		Schema: s,
		Raw:    tftypes.NewValue(terraformType, values),
	}
}

func TestManagedKubernetesServiceCIDROverlapValidator(t *testing.T) {
	ctx := context.Background()
	v := managedKubernetesServiceCIDROverlapValidator{}

	tests := map[string]struct {
		podCIDR     string
		serviceCIDR string
		wantErr     bool
	}{
		"disjoint":      {"10.244.0.0/16", "10.96.0.0/16", false},
		"overlapping":   {"10.244.0.0/16", "10.244.1.0/24", true},
		"identical":     {"10.96.0.0/16", "10.96.0.0/16", true},
		"pod unset":     {"", "10.96.0.0/16", false},
		"service unset": {"10.244.0.0/16", "", false},
		"neither set":   {"", "", false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			resp := &resource.ValidateConfigResponse{}
			v.ValidateResource(ctx, resource.ValidateConfigRequest{
				Config: managedKubernetesClusterConfig(t, ctx, tc.podCIDR, tc.serviceCIDR),
			}, resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Fatalf("got error=%v, want %v (diags: %v)", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}
