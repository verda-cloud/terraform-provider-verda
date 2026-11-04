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
	"fmt"
	"strings"
)

// parseImportID splits a composite import ID such as "pn_x/sn_y" into its
// parts. format names the parts for the error message, e.g. "<private_network_id>/<subnet_id>".
// Every part must be non-empty.
func parseImportID(id, format string) ([]string, error) {
	want := strings.Count(format, "/") + 1
	parts := strings.Split(id, "/")
	if len(parts) != want {
		return nil, fmt.Errorf("expected import ID in the form %s, got %q", format, id)
	}
	for _, p := range parts {
		if p == "" {
			return nil, fmt.Errorf("expected import ID in the form %s, got %q", format, id)
		}
	}
	return parts, nil
}
