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
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

const managedKubernetesNodePoolDefaultTimeout = 20 * time.Minute

var _ resource.Resource = &ManagedKubernetesNodePoolResource{}
var _ resource.ResourceWithImportState = &ManagedKubernetesNodePoolResource{}

func NewManagedKubernetesNodePoolResource() resource.Resource {
	return &ManagedKubernetesNodePoolResource{}
}

type ManagedKubernetesNodePoolResource struct {
	client *verda.Client
}

type ManagedKubernetesNodePoolResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	ClusterID     types.String   `tfsdk:"cluster_id"`
	Name          types.String   `tfsdk:"name"`
	InstanceType  types.String   `tfsdk:"instance_type"`
	Replicas      types.Int64    `tfsdk:"replicas"`
	SSHKeyIDs     types.Set      `tfsdk:"ssh_key_ids"`
	State         types.String   `tfsdk:"state"`
	ReadyReplicas types.Int64    `tfsdk:"ready_replicas"`
	Errors        types.List     `tfsdk:"errors"`
	CreatedAt     types.String   `tfsdk:"created_at"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func (r *ManagedKubernetesNodePoolResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_kubernetes_node_pool"
}

func (r *ManagedKubernetesNodePoolResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a node pool of a Verda managed Kubernetes cluster (experimental API)",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite resource ID in the form `<location>/<cluster-name>/<pool-name>`",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "Cluster that owns the pool, referenced by its composite ID `<location>/<cluster-name>` (e.g. `verda_managed_kubernetes_cluster.main.id`)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					managedKubernetesCompositeIDValidator{segments: 2},
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the node pool; a DNS-1123 label of at most 30 characters",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					managedKubernetesDNS1123LabelValidator{maxLen: 30},
				},
			},
			"instance_type": schema.StringAttribute{
				MarkdownDescription: "Verda VM type for the pool's workers (e.g. '1B200.30V')",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"replicas": schema.Int64Attribute{
				MarkdownDescription: "Desired number of worker nodes; 0 scales the pool down to zero",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"ssh_key_ids": schema.SetAttribute{
				MarkdownDescription: "SSH keys injected into each worker (at least one)",
				Required:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setRequiresReplaceModifier{},
				},
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "Lifecycle state of the pool: creating, active, degraded or deleting",
				Computed:            true,
			},
			"ready_replicas": schema.Int64Attribute{
				MarkdownDescription: "Number of workers currently ready",
				Computed:            true,
			},
			"errors": schema.ListNestedAttribute{
				MarkdownDescription: "API-reported reasons for a degraded pool, e.g. insufficient_capacity or insufficient_funds",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"reason": schema.StringAttribute{
							MarkdownDescription: "Machine-readable error reason",
							Computed:            true,
						},
						"message": schema.StringAttribute{
							MarkdownDescription: "Human-readable error message",
							Computed:            true,
						},
						"since": schema.StringAttribute{
							MarkdownDescription: "Timestamp when the error state began",
							Computed:            true,
						},
					},
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the pool was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// Create/Update wait for state active, Delete waits until the pool
			// is gone (20m default each).
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *ManagedKubernetesNodePoolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ManagedKubernetesNodePoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ManagedKubernetesNodePoolResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	location, clusterName, err := managedKubernetesParseClusterID(data.ClusterID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cluster_id", err.Error())
		return
	}

	var sshKeyIDs []string
	resp.Diagnostics.Append(data.SSHKeyIDs.ElementsAs(ctx, &sshKeyIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := data.Timeouts.Create(ctx, managedKubernetesNodePoolDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	replicas := int(data.Replicas.ValueInt64())
	pool, err := r.client.ManagedKubernetes.CreateNodePool(createCtx, location, clusterName, verda.CreateManagedKubernetesNodePoolRequest{
		Name:         data.Name.ValueString(),
		Replicas:     &replicas,
		InstanceType: data.InstanceType.ValueString(),
		SSHKeyIDs:    sshKeyIDs,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create managed Kubernetes node pool, got error: %s", err))
		return
	}

	// Persist the ID immediately: if the readiness wait fails, the pool is in
	// state as tainted instead of being duplicated on the next apply.
	data.ID = types.StringValue(managedKubernetesFormatNodePoolID(location, clusterName, pool.Name))
	r.flattenManagedKubernetesNodePoolToModel(ctx, pool, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pool, err = managedKubernetesWaitForNodePoolActive(createCtx, r.client.ManagedKubernetes, location, clusterName, pool.Name)
	if err != nil {
		resp.Diagnostics.AddError("Node pool not ready", fmt.Sprintf("Node pool %s was created but did not become active: %s. It is in state as tainted, so the next apply replaces it. To keep it instead, run `terraform untaint` on this resource before the next apply.", data.ID.ValueString(), err))
		return
	}

	r.flattenManagedKubernetesNodePoolToModel(ctx, pool, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedKubernetesNodePoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ManagedKubernetesNodePoolResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	location, clusterName, err := managedKubernetesParseClusterID(data.ClusterID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cluster_id", err.Error())
		return
	}

	pool, err := r.client.ManagedKubernetes.GetNodePool(ctx, location, clusterName, data.Name.ValueString())
	if managedKubernetesIsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read managed Kubernetes node pool, got error: %s", err))
		return
	}

	r.flattenManagedKubernetesNodePoolToModel(ctx, pool, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedKubernetesNodePoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ManagedKubernetesNodePoolResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	location, clusterName, err := managedKubernetesParseClusterID(data.ClusterID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cluster_id", err.Error())
		return
	}

	updateTimeout, diags := data.Timeouts.Update(ctx, managedKubernetesNodePoolDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateCtx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	replicas := int(data.Replicas.ValueInt64())
	pool, err := r.client.ManagedKubernetes.UpdateNodePool(updateCtx, location, clusterName, data.Name.ValueString(), verda.UpdateManagedKubernetesNodePoolRequest{
		Replicas: &replicas,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update managed Kubernetes node pool, got error: %s", err))
		return
	}

	pool, err = managedKubernetesWaitForNodePoolActive(updateCtx, r.client.ManagedKubernetes, location, clusterName, pool.Name)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error waiting for managed Kubernetes node pool update: %s", err))
		return
	}

	r.flattenManagedKubernetesNodePoolToModel(ctx, pool, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedKubernetesNodePoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ManagedKubernetesNodePoolResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	location, clusterName, err := managedKubernetesParseClusterID(data.ClusterID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cluster_id", err.Error())
		return
	}

	deleteTimeout, diags := data.Timeouts.Delete(ctx, managedKubernetesNodePoolDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteCtx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	err = r.client.ManagedKubernetes.DeleteNodePool(deleteCtx, location, clusterName, data.Name.ValueString())
	if err != nil {
		if managedKubernetesIsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete managed Kubernetes node pool, got error: %s", err))
		return
	}

	if err := managedKubernetesWaitForNodePoolDeleted(deleteCtx, r.client.ManagedKubernetes, location, clusterName, data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error waiting for managed Kubernetes node pool deletion: %s", err))
	}
}

func (r *ManagedKubernetesNodePoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	location, clusterName, poolName, err := managedKubernetesParseNodePoolID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), managedKubernetesFormatClusterID(location, clusterName))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), poolName)...)
}

var managedKubernetesNodePoolErrorObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"reason":  types.StringType,
		"message": types.StringType,
		"since":   types.StringType,
	},
}

func (r *ManagedKubernetesNodePoolResource) flattenManagedKubernetesNodePoolToModel(ctx context.Context, pool *verda.ManagedKubernetesNodePool, data *ManagedKubernetesNodePoolResourceModel, diagnostics *diag.Diagnostics) {
	location, clusterName, _ := managedKubernetesParseClusterID(data.ClusterID.ValueString())

	data.ID = types.StringValue(managedKubernetesFormatNodePoolID(location, clusterName, pool.Name))
	data.Name = types.StringValue(pool.Name)
	data.InstanceType = types.StringValue(pool.InstanceType)
	data.Replicas = types.Int64Value(int64(pool.Replicas))
	data.ReadyReplicas = types.Int64Value(int64(pool.ReadyReplicas))
	data.State = types.StringValue(pool.State)
	data.CreatedAt = types.StringValue(pool.CreatedAt.Format("2006-01-02T15:04:05Z"))

	// The API may omit ssh_key_ids in responses; keep the configured value.
	if len(pool.SSHKeyIDs) > 0 {
		sshKeySet, diags := types.SetValueFrom(ctx, types.StringType, pool.SSHKeyIDs)
		diagnostics.Append(diags...)
		data.SSHKeyIDs = sshKeySet
	}

	if len(pool.Errors) > 0 {
		elements := make([]attr.Value, 0, len(pool.Errors))
		for _, e := range pool.Errors {
			element, diags := types.ObjectValue(managedKubernetesNodePoolErrorObjectType.AttrTypes, map[string]attr.Value{
				"reason":  types.StringValue(e.Reason),
				"message": types.StringValue(e.Message),
				"since":   types.StringValue(e.Since.Format("2006-01-02T15:04:05Z")),
			})
			diagnostics.Append(diags...)
			elements = append(elements, element)
		}
		errorsList, diags := types.ListValue(managedKubernetesNodePoolErrorObjectType, elements)
		diagnostics.Append(diags...)
		data.Errors = errorsList
	} else {
		data.Errors = types.ListNull(managedKubernetesNodePoolErrorObjectType)
	}
}
