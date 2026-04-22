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
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

var _ resource.Resource = &VolumeAttachmentResource{}
var _ resource.ResourceWithImportState = &VolumeAttachmentResource{}

func NewVolumeAttachmentResource() resource.Resource {
	return &VolumeAttachmentResource{}
}

type VolumeAttachmentResource struct {
	client *verda.Client
}

type VolumeAttachmentResourceModel struct {
	VolumeID                 types.String `tfsdk:"volume_id"`
	InstanceID               types.String `tfsdk:"instance_id"`
	MountCommand             types.String `tfsdk:"mount_command"`
	CreateDirectoryCommand   types.String `tfsdk:"create_directory_command"`
	FilesystemToFstabCommand types.String `tfsdk:"filesystem_to_fstab_command"`
}

func (r *VolumeAttachmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_attachment"
}

func (r *VolumeAttachmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Attaches a Verda volume to an instance. Mount commands are available as outputs after attachment.",

		Attributes: map[string]schema.Attribute{
			"volume_id": schema.StringAttribute{
				MarkdownDescription: "ID of the volume to attach",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "ID of the instance to attach the volume to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mount_command": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Shell command to mount the volume on the instance",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"create_directory_command": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Shell command to create the mount directory on the instance",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"filesystem_to_fstab_command": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Shell command to add the volume to /etc/fstab for persistent mounts",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *VolumeAttachmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VolumeAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VolumeAttachmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := waitForInstanceIP(ctx, r.client, data.InstanceID.ValueString(), 3*time.Minute); err != nil {
		resp.Diagnostics.AddError("Instance Not Ready", err.Error())
		return
	}

	err := r.client.Volumes.AttachVolume(ctx, data.VolumeID.ValueString(), verda.VolumeAttachRequest{
		InstanceID: data.InstanceID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to attach volume, got error: %s", err))
		return
	}

	volume, err := r.client.Volumes.GetVolume(ctx, data.VolumeID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume after attachment, got error: %s", err))
		return
	}

	flattenVolumeAttachmentToModel(volume, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VolumeAttachmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	volume, err := r.client.Volumes.GetVolume(ctx, data.VolumeID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume, got error: %s", err))
		return
	}

	// If the volume is no longer attached to this instance, remove from state
	if volume.InstanceID == nil || *volume.InstanceID != data.InstanceID.ValueString() {
		resp.State.RemoveResource(ctx)
		return
	}

	flattenVolumeAttachmentToModel(volume, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Volume attachments cannot be updated. Changes require replacing the resource.",
	)
}

func (r *VolumeAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VolumeAttachmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Volumes.DetachVolume(ctx, data.VolumeID.ValueString(), verda.VolumeDetachRequest{
		InstanceID: data.InstanceID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to detach volume, got error: %s", err))
		return
	}
}

func (r *VolumeAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected import ID in the format: {volume_id}/{instance_id}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("volume_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), parts[1])...)
}

func waitForInstanceIP(ctx context.Context, client *verda.Client, instanceID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		instance, err := client.Instances.GetByID(ctx, instanceID)
		if err != nil {
			return fmt.Errorf("unable to read instance while waiting for IP address: %w", err)
		}
		if instance.IP != nil && *instance.IP != "" {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for instance IP address")
		case <-time.After(10 * time.Second):
		}
	}
	return fmt.Errorf("instance %s did not receive an IP address within %s", instanceID, timeout)
}

func flattenVolumeAttachmentToModel(volume *verda.Volume, data *VolumeAttachmentResourceModel) {
	if volume.MountCommand != nil {
		data.MountCommand = types.StringValue(*volume.MountCommand)
	} else {
		data.MountCommand = types.StringNull()
	}

	if volume.CreateDirectoryCommand != nil {
		data.CreateDirectoryCommand = types.StringValue(*volume.CreateDirectoryCommand)
	} else {
		data.CreateDirectoryCommand = types.StringNull()
	}

	if volume.FilesystemToFstabCommand != nil {
		data.FilesystemToFstabCommand = types.StringValue(*volume.FilesystemToFstabCommand)
	} else {
		data.FilesystemToFstabCommand = types.StringNull()
	}
}
