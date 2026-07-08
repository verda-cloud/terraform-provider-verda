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
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPreserveKnownImage(t *testing.T) {
	image := types.StringValue("79091d37-cf39-43cd-b193-ba9c2611c988")
	data := InstanceResourceModel{
		Image: types.StringValue("ubuntu-24.04"),
	}

	preserveKnownImage(image, &data)

	if data.Image.ValueString() != image.ValueString() {
		t.Fatalf("expected image %q, got %q", image.ValueString(), data.Image.ValueString())
	}
}
