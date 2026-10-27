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
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

const (
	managedKubernetesKubeconfigDefaultCreateTimeout = 5 * time.Minute
	managedKubernetesKubeconfigDefaultDeleteTimeout = 10 * time.Minute
)

var _ resource.Resource = &ManagedKubernetesKubeconfigResource{}
var _ resource.ResourceWithImportState = &ManagedKubernetesKubeconfigResource{}

func NewManagedKubernetesKubeconfigResource() resource.Resource {
	return &ManagedKubernetesKubeconfigResource{}
}

type ManagedKubernetesKubeconfigResource struct {
	client *verda.Client
}

type ManagedKubernetesKubeconfigResourceModel struct {
	ID          types.String   `tfsdk:"id"`
	ClusterID   types.String   `tfsdk:"cluster_id"`
	CommonName  types.String   `tfsdk:"common_name"`
	Groups      types.List     `tfsdk:"groups"`
	DisplayName types.String   `tfsdk:"display_name"`
	Duration    types.String   `tfsdk:"duration"`
	Status      types.String   `tfsdk:"status"`
	NotAfter    types.String   `tfsdk:"not_after"`
	Content     types.String   `tfsdk:"content"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
}

func (r *ManagedKubernetesKubeconfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_kubernetes_kubeconfig"
}

func (r *ManagedKubernetesKubeconfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Issues a client-certificate kubeconfig for a Verda managed Kubernetes cluster (experimental API). Any change re-issues the certificate by replacing the resource. Deletion does not revoke already-downloaded kubeconfigs; they keep working until not_after. Rotate by tainting the resource and re-applying.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-generated certificate ID",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "Cluster that owns the kubeconfig, referenced by its composite ID `<location>/<cluster-name>` (e.g. `verda_managed_kubernetes_cluster.main.id`)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					managedKubernetesCompositeIDValidator{segments: 2},
				},
			},
			"common_name": schema.StringAttribute{
				MarkdownDescription: "Certificate Common Name identifying the user; control-plane identities (system:*) are rejected by the API",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"groups": schema.ListAttribute{
				MarkdownDescription: "Certificate group memberships; system:masters grants cluster-admin access. A kubeconfig without groups authenticates but authorizes nothing",
				Required:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-facing label for the kubeconfig (at most 128 characters)",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(128),
				},
			},
			"duration": schema.StringAttribute{
				MarkdownDescription: "Certificate lifetime in Go duration syntax (e.g. \"720h\"); minimum 1h, defaults to 90 days when omitted",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					managedKubernetesKubeconfigDurationValidator{},
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Issuance status of the kubeconfig: pending or ready",
				Computed:            true,
			},
			"not_after": schema.StringAttribute{
				MarkdownDescription: "Certificate expiry timestamp",
				Computed:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "Rendered kubeconfig YAML, available once status is ready",
				Computed:            true,
				Sensitive:           true,
			},
			// Create waits for status ready, Delete waits until the record is
			// gone (5m / 10m defaults).
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Delete: true,
			}),
		},
	}
}

func (r *ManagedKubernetesKubeconfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ManagedKubernetesKubeconfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ManagedKubernetesKubeconfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	location, clusterName, err := managedKubernetesParseClusterID(data.ClusterID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cluster_id", err.Error())
		return
	}

	var groups []string
	if !data.Groups.IsNull() && !data.Groups.IsUnknown() {
		resp.Diagnostics.Append(data.Groups.ElementsAs(ctx, &groups, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	createReq := verda.CreateManagedKubernetesKubeconfigRequest{
		CommonName: data.CommonName.ValueString(),
		Groups:     groups,
	}
	if !data.DisplayName.IsNull() {
		createReq.DisplayName = data.DisplayName.ValueString()
	}
	if !data.Duration.IsNull() {
		createReq.Duration = data.Duration.ValueString()
	}

	createTimeout, diags := data.Timeouts.Create(ctx, managedKubernetesKubeconfigDefaultCreateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	kubeconfig, err := r.client.ManagedKubernetes.CreateKubeconfig(createCtx, location, clusterName, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create managed Kubernetes kubeconfig, got error: %s", err))
		return
	}

	// Persist the server ID immediately: if the readiness wait fails, the
	// record can still be located (and destroyed) on the next apply.
	data.ID = types.StringValue(kubeconfig.ID)
	r.flattenManagedKubernetesKubeconfigToModel(ctx, kubeconfig, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	kubeconfig, err = managedKubernetesWaitForKubeconfigReady(createCtx, r.client.ManagedKubernetes, location, clusterName, kubeconfig.ID)
	if err != nil {
		resp.Diagnostics.AddError("Kubeconfig not ready", fmt.Sprintf("Kubeconfig %s was created but did not become ready: %s. It is in state as tainted, so the next apply replaces it. To keep it instead, run `terraform untaint` on this resource before the next apply.", data.ID.ValueString(), err))
		return
	}

	r.flattenManagedKubernetesKubeconfigToModel(ctx, kubeconfig, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedKubernetesKubeconfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ManagedKubernetesKubeconfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	location, clusterName, err := managedKubernetesParseClusterID(data.ClusterID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cluster_id", err.Error())
		return
	}

	kubeconfig, err := r.client.ManagedKubernetes.GetKubeconfig(ctx, location, clusterName, data.ID.ValueString())
	if managedKubernetesIsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read managed Kubernetes kubeconfig, got error: %s", err))
		return
	}

	r.flattenManagedKubernetesKubeconfigToModel(ctx, kubeconfig, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Every configurable attribute forces replacement, so a real update never
// reaches this method; the framework pre-seeds resp.State from req.Plan, so
// an empty body still returns valid state if it were ever called.
func (r *ManagedKubernetesKubeconfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *ManagedKubernetesKubeconfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ManagedKubernetesKubeconfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	location, clusterName, err := managedKubernetesParseClusterID(data.ClusterID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cluster_id", err.Error())
		return
	}

	deleteTimeout, diags := data.Timeouts.Delete(ctx, managedKubernetesKubeconfigDefaultDeleteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteCtx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	err = r.client.ManagedKubernetes.DeleteKubeconfig(deleteCtx, location, clusterName, data.ID.ValueString())
	if err != nil {
		if managedKubernetesIsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete managed Kubernetes kubeconfig, got error: %s", err))
		return
	}

	if err := managedKubernetesWaitForKubeconfigDeleted(deleteCtx, r.client.ManagedKubernetes, location, clusterName, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error waiting for managed Kubernetes kubeconfig deletion: %s", err))
	}
}

func (r *ManagedKubernetesKubeconfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	location, clusterName, id, err := managedKubernetesParseKubeconfigImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), managedKubernetesFormatClusterID(location, clusterName))...)
}

func (r *ManagedKubernetesKubeconfigResource) flattenManagedKubernetesKubeconfigToModel(ctx context.Context, kubeconfig *verda.ManagedKubernetesKubeconfig, data *ManagedKubernetesKubeconfigResourceModel, diagnostics *diag.Diagnostics) {
	data.ID = types.StringValue(kubeconfig.ID)
	data.CommonName = types.StringValue(kubeconfig.CommonName)
	data.Status = types.StringValue(kubeconfig.Status)

	// The API may omit any of these; keep the configured value then.
	if len(kubeconfig.Groups) > 0 {
		groups, diags := types.ListValueFrom(ctx, types.StringType, kubeconfig.Groups)
		diagnostics.Append(diags...)
		data.Groups = groups
	}
	if kubeconfig.DisplayName != "" {
		data.DisplayName = types.StringValue(kubeconfig.DisplayName)
	}

	// Content and NotAfter only exist once the record is ready; while
	// pending, keep whatever is already in state.
	if kubeconfig.Kubeconfig != "" {
		data.Content = types.StringValue(kubeconfig.Kubeconfig)
	}
	if kubeconfig.NotAfter != nil {
		data.NotAfter = types.StringValue(kubeconfig.NotAfter.Format("2006-01-02T15:04:05Z"))
	}
}
