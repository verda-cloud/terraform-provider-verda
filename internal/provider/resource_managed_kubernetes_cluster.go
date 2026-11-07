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
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

const managedKubernetesClusterDefaultTimeout = 30 * time.Minute

var _ resource.Resource = &ManagedKubernetesClusterResource{}
var _ resource.ResourceWithConfigValidators = &ManagedKubernetesClusterResource{}
var _ resource.ResourceWithImportState = &ManagedKubernetesClusterResource{}

func NewManagedKubernetesClusterResource() resource.Resource {
	return &ManagedKubernetesClusterResource{}
}

type ManagedKubernetesClusterResource struct {
	client *verda.Client
}

type ManagedKubernetesClusterResourceModel struct {
	ID                types.String   `tfsdk:"id"`
	Name              types.String   `tfsdk:"name"`
	KubernetesVersion types.String   `tfsdk:"kubernetes_version"`
	Location          types.String   `tfsdk:"location"`
	NetworkID         types.String   `tfsdk:"network_id"`
	SubnetID          types.String   `tfsdk:"subnet_id"`
	PodCIDR           types.String   `tfsdk:"pod_cidr"`
	ServiceCIDR       types.String   `tfsdk:"service_cidr"`
	DNSSuffix         types.String   `tfsdk:"dns_suffix"`
	Phase             types.String   `tfsdk:"phase"`
	ExternalEndpoint  types.String   `tfsdk:"external_endpoint"`
	CreatedAt         types.String   `tfsdk:"created_at"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
}

func (r *ManagedKubernetesClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_kubernetes_cluster"
}

func (r *ManagedKubernetesClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Verda managed Kubernetes cluster (experimental API)",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite resource ID in the form `<location>/<cluster-name>`",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the cluster; a DNS-1123 label of at most 30 characters",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					managedKubernetesDNS1123LabelValidator{maxLen: 30},
				},
			},
			"kubernetes_version": schema.StringAttribute{
				MarkdownDescription: "Kubernetes version of the cluster; must exist in the version catalogue. Updating runs a rolling control-plane upgrade",
				Required:            true,
			},
			"location": schema.StringAttribute{
				MarkdownDescription: "Location code for the cluster (e.g. 'FIN-03'); must match the subnet's location",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "ID of the project's private network",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subnet_id": schema.StringAttribute{
				MarkdownDescription: "ID of the subnet within the private network; its CIDR must not overlap pod_cidr or service_cidr",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"pod_cidr": schema.StringAttribute{
				MarkdownDescription: "Pod IP range; must be /21 or larger and must not overlap service_cidr",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					managedKubernetesCIDRValidator{},
					managedKubernetesPodCIDRBitsValidator{},
				},
			},
			"service_cidr": schema.StringAttribute{
				MarkdownDescription: "Service ClusterIP range; must not overlap pod_cidr",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					managedKubernetesCIDRValidator{},
				},
			},
			"dns_suffix": schema.StringAttribute{
				MarkdownDescription: "DNS suffix of the cluster",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"phase": schema.StringAttribute{
				MarkdownDescription: "Lifecycle phase of the cluster: provisioning, running, degraded or deleting",
				Computed:            true,
			},
			"external_endpoint": schema.StringAttribute{
				MarkdownDescription: "External API endpoint of the cluster",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the cluster was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// Create/Update wait for phase running, Delete waits until the
			// cluster is gone (30m default each).
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *ManagedKubernetesClusterResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		managedKubernetesServiceCIDROverlapValidator{},
	}
}

func (r *ManagedKubernetesClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*verda.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *verda.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ManagedKubernetesClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ManagedKubernetesClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := data.Timeouts.Create(ctx, managedKubernetesClusterDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	cluster, err := r.client.ManagedKubernetes.Create(createCtx, verda.CreateManagedKubernetesClusterRequest{
		Name:              data.Name.ValueString(),
		KubernetesVersion: data.KubernetesVersion.ValueString(),
		PodCIDR:           data.PodCIDR.ValueString(),
		ServiceCIDR:       data.ServiceCIDR.ValueString(),
		Location:          data.Location.ValueString(),
		NetworkID:         data.NetworkID.ValueString(),
		SubnetID:          data.SubnetID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create managed Kubernetes cluster, got error: %s", err))
		return
	}

	// Persist the ID immediately: if the readiness wait fails, the cluster is
	// in state as tainted instead of being duplicated on the next apply.
	data.ID = types.StringValue(managedKubernetesFormatClusterID(cluster.Location, cluster.Name))
	r.flattenManagedKubernetesClusterToModel(cluster, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cluster, err = managedKubernetesWaitForClusterRunning(createCtx, r.client.ManagedKubernetes, cluster.Location, cluster.Name)
	if err != nil {
		resp.Diagnostics.AddError("Cluster not ready", fmt.Sprintf("Cluster %s was created but did not become running: %s. It is in state as tainted, so the next apply replaces it. To keep it instead, run `terraform untaint` on this resource before the next apply.", data.ID.ValueString(), err))
		return
	}

	r.flattenManagedKubernetesClusterToModel(cluster, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedKubernetesClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ManagedKubernetesClusterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	cluster, err := r.client.ManagedKubernetes.Get(ctx, data.Location.ValueString(), data.Name.ValueString())
	if managedKubernetesIsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read managed Kubernetes cluster, got error: %s", err))
		return
	}

	r.flattenManagedKubernetesClusterToModel(cluster, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedKubernetesClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ManagedKubernetesClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := data.Timeouts.Update(ctx, managedKubernetesClusterDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateCtx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	cluster, err := r.client.ManagedKubernetes.Update(updateCtx, data.Location.ValueString(), data.Name.ValueString(), verda.UpdateManagedKubernetesClusterRequest{
		KubernetesVersion: data.KubernetesVersion.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update managed Kubernetes cluster, got error: %s", err))
		return
	}

	cluster, err = managedKubernetesWaitForClusterRunning(updateCtx, r.client.ManagedKubernetes, cluster.Location, cluster.Name)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error waiting for managed Kubernetes cluster update: %s", err))
		return
	}

	r.flattenManagedKubernetesClusterToModel(cluster, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedKubernetesClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ManagedKubernetesClusterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := data.Timeouts.Delete(ctx, managedKubernetesClusterDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteCtx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	err := r.client.ManagedKubernetes.Delete(deleteCtx, data.Location.ValueString(), data.Name.ValueString())
	if err != nil {
		if managedKubernetesIsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete managed Kubernetes cluster, got error: %s", err))
		return
	}

	if err := managedKubernetesWaitForClusterDeleted(deleteCtx, r.client.ManagedKubernetes, data.Location.ValueString(), data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error waiting for managed Kubernetes cluster deletion: %s", err))
	}
}

func (r *ManagedKubernetesClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	location, name, err := managedKubernetesParseClusterID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("location"), location)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
}

func (r *ManagedKubernetesClusterResource) flattenManagedKubernetesClusterToModel(cluster *verda.ManagedKubernetesCluster, data *ManagedKubernetesClusterResourceModel) {
	data.ID = types.StringValue(managedKubernetesFormatClusterID(cluster.Location, cluster.Name))
	data.Name = types.StringValue(cluster.Name)
	data.KubernetesVersion = types.StringValue(cluster.KubernetesVersion)
	data.Location = types.StringValue(cluster.Location)
	data.NetworkID = types.StringValue(cluster.NetworkID)
	data.SubnetID = types.StringValue(cluster.SubnetID)
	data.PodCIDR = types.StringValue(cluster.PodCIDR)
	data.ServiceCIDR = types.StringValue(cluster.ServiceCIDR)
	data.DNSSuffix = types.StringValue(cluster.DNSSuffix)
	data.Phase = types.StringValue(cluster.Phase)
	data.ExternalEndpoint = types.StringValue(cluster.ExternalEndpoint)
	data.CreatedAt = types.StringValue(cluster.CreatedAt.Format("2006-01-02T15:04:05Z"))
}
