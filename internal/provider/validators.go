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
