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
	"errors"
	"fmt"
	"testing"

	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

func TestAPIStatus(t *testing.T) {
	notFound := &verda.APIError{StatusCode: 404, Message: "not found"}
	conflict := &verda.APIError{StatusCode: 409, Message: "subnet still has 1 attached instance(s)"}

	tests := map[string]struct {
		err          error
		wantStatus   int
		wantNotFound bool
		wantConflict bool
	}{
		"nil":                {err: nil},
		"plain error":        {err: errors.New("API error 404: not found")},
		"404":                {err: notFound, wantStatus: 404, wantNotFound: true},
		"409":                {err: conflict, wantStatus: 409, wantConflict: true},
		"wrapped 404":        {err: fmt.Errorf("get subnet: %w", notFound), wantStatus: 404, wantNotFound: true},
		"500":                {err: &verda.APIError{StatusCode: 500}, wantStatus: 500},
		"validation error":   {err: &verda.ValidationError{Field: "name"}},
		"double-wrapped 409": {err: fmt.Errorf("a: %w", fmt.Errorf("b: %w", conflict)), wantStatus: 409, wantConflict: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := apiStatus(tt.err); got != tt.wantStatus {
				t.Errorf("apiStatus() = %d, want %d", got, tt.wantStatus)
			}
			if got := isNotFound(tt.err); got != tt.wantNotFound {
				t.Errorf("isNotFound() = %v, want %v", got, tt.wantNotFound)
			}
			if got := isConflict(tt.err); got != tt.wantConflict {
				t.Errorf("isConflict() = %v, want %v", got, tt.wantConflict)
			}
		})
	}
}

func TestParseImportID(t *testing.T) {
	tests := map[string]struct {
		id      string
		format  string
		want    []string
		wantErr bool
	}{
		"subnet":            {id: "pn_a/sn_b", format: "<private_network_id>/<subnet_id>", want: []string{"pn_a", "sn_b"}},
		"route":             {id: "pn_a/sn_b/rt_c", format: "<private_network_id>/<subnet_id>/<route_id>", want: []string{"pn_a", "sn_b", "rt_c"}},
		"default subnet":    {id: "pn_a/FIN-03", format: "<private_network_id>/<location>", want: []string{"pn_a", "FIN-03"}},
		"single part":       {id: "pn_a", format: "<private_network_id>", want: []string{"pn_a"}},
		"too few parts":     {id: "pn_a", format: "<private_network_id>/<subnet_id>", wantErr: true},
		"too many parts":    {id: "pn_a/sn_b/rt_c", format: "<private_network_id>/<subnet_id>", wantErr: true},
		"empty part":        {id: "pn_a/", format: "<private_network_id>/<subnet_id>", wantErr: true},
		"leading slash":     {id: "/sn_b", format: "<private_network_id>/<subnet_id>", wantErr: true},
		"empty id":          {id: "", format: "<private_network_id>", wantErr: true},
		"empty middle part": {id: "pn_a//rt_c", format: "<private_network_id>/<subnet_id>/<route_id>", wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := parseImportID(tt.id, tt.format)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
