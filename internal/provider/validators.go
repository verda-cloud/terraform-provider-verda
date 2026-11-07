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
	"net/netip"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// volumeMountValidator validates that volume mount fields are appropriate for the volume type
type volumeMountValidator struct{}

func (v volumeMountValidator) Description(ctx context.Context) string {
	return "Validates that volume mount fields match the volume type"
}

func (v volumeMountValidator) MarkdownDescription(ctx context.Context) string {
	return "Validates that volume mount fields match the volume type"
}

func (v volumeMountValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	// If the object is null or unknown, no validation needed
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	// Get the object attributes
	attrs := req.ConfigValue.Attributes()

	// Get the type field
	typeVal, ok := attrs["type"].(types.String)
	if !ok || typeVal.IsNull() || typeVal.IsUnknown() {
		return
	}

	volumeType := typeVal.ValueString()

	// Get the optional fields
	secretName, _ := attrs["secret_name"].(types.String)
	sizeInMB, _ := attrs["size_in_mb"].(types.Int64)
	volumeID, _ := attrs["volume_id"].(types.String)

	// Validate based on type
	switch volumeType {
	case "scratch":
		// scratch volumes can optionally have size_in_mb
		if !secretName.IsNull() && secretName.ValueString() != "" {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("secret_name"),
				"Invalid Field for Volume Type",
				fmt.Sprintf("secret_name can only be specified for 'secret' type volumes, not '%s'", volumeType),
			)
		}
		if !volumeID.IsNull() && volumeID.ValueString() != "" {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("volume_id"),
				"Invalid Field for Volume Type",
				fmt.Sprintf("volume_id can only be specified for 'shared' type volumes, not '%s'", volumeType),
			)
		}

	case "memory":
		// memory volumes can optionally have size_in_mb
		if !secretName.IsNull() && secretName.ValueString() != "" {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("secret_name"),
				"Invalid Field for Volume Type",
				fmt.Sprintf("secret_name can only be specified for 'secret' type volumes, not '%s'", volumeType),
			)
		}
		if !volumeID.IsNull() && volumeID.ValueString() != "" {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("volume_id"),
				"Invalid Field for Volume Type",
				fmt.Sprintf("volume_id can only be specified for 'shared' type volumes, not '%s'", volumeType),
			)
		}

	case "secret":
		// secret volumes must have secret_name
		if !sizeInMB.IsNull() {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("size_in_mb"),
				"Invalid Field for Volume Type",
				fmt.Sprintf("size_in_mb can only be specified for 'scratch' or 'memory' type volumes, not '%s'", volumeType),
			)
		}
		if !volumeID.IsNull() && volumeID.ValueString() != "" {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("volume_id"),
				"Invalid Field for Volume Type",
				fmt.Sprintf("volume_id can only be specified for 'shared' type volumes, not '%s'", volumeType),
			)
		}
		// Require secret_name for secret type
		// Skip validation if value is unknown (e.g., referencing another resource that hasn't been created yet)
		if !secretName.IsUnknown() && (secretName.IsNull() || secretName.ValueString() == "") {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("secret_name"),
				"Missing Required Field",
				"secret_name is required for 'secret' type volumes",
			)
		}

	case "shared":
		// shared volumes must have volume_id
		if !secretName.IsNull() && secretName.ValueString() != "" {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("secret_name"),
				"Invalid Field for Volume Type",
				fmt.Sprintf("secret_name can only be specified for 'secret' type volumes, not '%s'", volumeType),
			)
		}
		if !sizeInMB.IsNull() {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("size_in_mb"),
				"Invalid Field for Volume Type",
				fmt.Sprintf("size_in_mb can only be specified for 'scratch' or 'memory' type volumes, not '%s'", volumeType),
			)
		}
		// Require volume_id for shared type
		// Skip validation if value is unknown (e.g., referencing another resource that hasn't been created yet)
		if !volumeID.IsUnknown() && (volumeID.IsNull() || volumeID.ValueString() == "") {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("volume_id"),
				"Missing Required Field",
				"volume_id is required for 'shared' type volumes",
			)
		}

	default:
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("type"),
			"Invalid Volume Type",
			fmt.Sprintf("Volume type must be 'scratch', 'memory', 'secret', or 'shared', got: '%s'", volumeType),
		)
	}
}

// osVolumeOnDestroyValidator accepts the three OS volume policies.
type osVolumeOnDestroyValidator struct{}

func (v osVolumeOnDestroyValidator) Description(ctx context.Context) string {
	return "on_destroy must be one of delete_permanently, move_to_trash, keep_detached"
}

func (v osVolumeOnDestroyValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v osVolumeOnDestroyValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	switch req.ConfigValue.ValueString() {
	case osVolumeDeletePermanently, osVolumeMoveToTrash, osVolumeKeepDetached:
		return
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid on_destroy",
		fmt.Sprintf("%q is not one of %q, %q, %q", req.ConfigValue.ValueString(), osVolumeDeletePermanently, osVolumeMoveToTrash, osVolumeKeepDetached))
}

var managedKubernetesDNS1123LabelRegexp = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// managedKubernetesDNS1123LabelValidator validates DNS-1123 label names used
// for managed Kubernetes cluster and node pool names (max 30 chars).
type managedKubernetesDNS1123LabelValidator struct {
	maxLen int
}

func (v managedKubernetesDNS1123LabelValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("value must be a DNS-1123 label of at most %d characters", v.maxLen)
}

func (v managedKubernetesDNS1123LabelValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v managedKubernetesDNS1123LabelValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	if len(s) == 0 || len(s) > v.maxLen || !managedKubernetesDNS1123LabelRegexp.MatchString(s) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid name",
			fmt.Sprintf("%q must be a DNS-1123 label of 1-%d characters: lowercase alphanumerics and '-', starting and ending with an alphanumeric", s, v.maxLen))
	}
}

// managedKubernetesCIDRValidator validates an IPv4 CIDR prefix.
type managedKubernetesCIDRValidator struct{}

func (v managedKubernetesCIDRValidator) Description(ctx context.Context) string {
	return "value must be an IPv4 CIDR prefix"
}

func (v managedKubernetesCIDRValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v managedKubernetesCIDRValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	prefix, err := netip.ParsePrefix(s)
	if err != nil || !prefix.Addr().Is4() {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid CIDR",
			fmt.Sprintf("%q must be an IPv4 CIDR prefix, e.g. \"10.244.0.0/16\"", s))
	}
}

// managedKubernetesPodCIDRBitsValidator rejects pod CIDRs smaller than /21,
// matching the API rule. Syntax errors are left to managedKubernetesCIDRValidator.
type managedKubernetesPodCIDRBitsValidator struct{}

func (v managedKubernetesPodCIDRBitsValidator) Description(ctx context.Context) string {
	return "pod_cidr must be /21 or larger"
}

func (v managedKubernetesPodCIDRBitsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v managedKubernetesPodCIDRBitsValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	prefix, err := netip.ParsePrefix(s)
	if err != nil {
		return
	}
	if prefix.Bits() > 21 {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid pod_cidr",
			fmt.Sprintf("pod_cidr must be /21 or larger, got %q", s))
	}
}

// managedKubernetesKubeconfigDurationValidator validates a cert lifetime in
// Go duration syntax (e.g. "720h"), minimum 1 hour.
type managedKubernetesKubeconfigDurationValidator struct{}

func (v managedKubernetesKubeconfigDurationValidator) Description(ctx context.Context) string {
	return "duration must be in Go duration syntax and at least 1h"
}

func (v managedKubernetesKubeconfigDurationValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v managedKubernetesKubeconfigDurationValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	d, err := time.ParseDuration(s)
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid duration",
			fmt.Sprintf("%q is not a valid Go duration (e.g. \"720h\")", s))
		return
	}
	if d < time.Hour {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid duration",
			fmt.Sprintf("duration must be at least 1h, got %q", s))
	}
}

// managedKubernetesCompositeIDValidator validates slash-joined composite IDs
// such as the cluster_id reference ("<location>/<cluster-name>").
type managedKubernetesCompositeIDValidator struct {
	segments int
}

func (v managedKubernetesCompositeIDValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("value must consist of %d slash-separated segments", v.segments)
}

func (v managedKubernetesCompositeIDValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v managedKubernetesCompositeIDValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	parts := strings.Split(s, "/")
	if len(parts) != v.segments {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid ID format",
			fmt.Sprintf("%q must consist of %d segments joined by '/' (e.g. \"FIN-03/my-cluster\"), got %d", s, v.segments, len(parts)))
		return
	}
	for _, p := range parts {
		if p == "" {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid ID format",
				fmt.Sprintf("%q must not contain empty segments", s))
			return
		}
	}
}

// managedKubernetesCIDRsOverlap reports whether two CIDR strings parse into
// overlapping prefixes; unparsable input is left to the CIDR validators.
func managedKubernetesCIDRsOverlap(a, b string) bool {
	pa, errA := netip.ParsePrefix(a)
	pb, errB := netip.ParsePrefix(b)
	if errA != nil || errB != nil {
		return false
	}
	return pa.Overlaps(pb)
}

// managedKubernetesServiceCIDROverlapValidator is a resource-level validator
// on the managed Kubernetes cluster rejecting overlapping pod/service CIDRs.
type managedKubernetesServiceCIDROverlapValidator struct{}

func (v managedKubernetesServiceCIDROverlapValidator) Description(ctx context.Context) string {
	return "service_cidr must not overlap pod_cidr"
}

func (v managedKubernetesServiceCIDROverlapValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v managedKubernetesServiceCIDROverlapValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var pod, service types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("pod_cidr"), &pod)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("service_cidr"), &service)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if pod.IsNull() || pod.IsUnknown() || service.IsNull() || service.IsUnknown() {
		return
	}
	if managedKubernetesCIDRsOverlap(pod.ValueString(), service.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("service_cidr"), "Overlapping CIDRs",
			"service_cidr must not overlap pod_cidr")
	}
}
