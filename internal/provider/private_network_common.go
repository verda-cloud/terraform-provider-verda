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
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

// defaultPrivateNetworkDeleteTimeout bounds the 409 retry loop of private network and subnet deletes.
const defaultPrivateNetworkDeleteTimeout = 10 * time.Minute

// Backoff for deleteRetryingConflicts; tests shorten them.
var (
	privateNetworkDeleteRetryInitial = 5 * time.Second
	privateNetworkDeleteRetryMax     = 30 * time.Second
)

// privateNetworkNameValidators enforce the API's 1–64 character limit on private network and subnet names.
var privateNetworkNameValidators = []validator.String{stringvalidator.LengthBetween(1, 64)}

// deleteRetryingConflicts calls del until it succeeds, retrying while the API
// answers 409. Instance deletion is asynchronous, so a private network or subnet delete
// issued right after its instances are destroyed is refused until they are
// gone. A 404 counts as success: deleting a private network cascades to its subnets.
// The caller sets the deadline on ctx; on timeout the last API error, which
// names the attached instances, is returned.
func deleteRetryingConflicts(ctx context.Context, del func(context.Context) error) error {
	wait := privateNetworkDeleteRetryInitial
	for {
		err := del(ctx)
		if err == nil || isNotFound(err) {
			return nil
		}
		if !isConflict(err) {
			return err
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("still refused when the delete timeout ran out: %w", err)
			}
			return ctx.Err()
		case <-time.After(wait):
		}
		wait = min(wait*2, privateNetworkDeleteRetryMax)
	}
}

// formatTimestamp renders an API timestamp the way the provider stores them.
func formatTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// findPrivateNetwork picks one network from a listing by name, or the project default
// when name is empty. It errors unless exactly one network matches.
func findPrivateNetwork(networks []verda.PrivateNetwork, name string) (*verda.PrivateNetwork, error) {
	var found []verda.PrivateNetwork
	for _, n := range networks {
		if (name != "" && n.Name == name) || (name == "" && n.IsDefault) {
			found = append(found, n)
		}
	}
	what := fmt.Sprintf("named %q", name)
	if name == "" {
		what = "marked as the project default"
	}
	switch len(found) {
	case 1:
		return &found[0], nil
	case 0:
		return nil, fmt.Errorf("no private network %s", what)
	default:
		return nil, fmt.Errorf("%d private networks %s", len(found), what)
	}
}

// findSubnet picks one subnet from a private network's listing by name or, when name is
// empty, the default subnet of location. A non-empty location also filters
// name lookups. It errors unless exactly one subnet matches.
func findSubnet(subnets []verda.Subnet, name, location string) (*verda.Subnet, error) {
	var found []verda.Subnet
	for _, s := range subnets {
		if location != "" && s.Location.Code != location {
			continue
		}
		if (name != "" && s.Name == name) || (name == "" && s.IsDefault) {
			found = append(found, s)
		}
	}
	what := fmt.Sprintf("named %q", name)
	if name == "" {
		what = "marked as the default"
	}
	if location != "" {
		what += " in " + location
	}
	switch len(found) {
	case 1:
		return &found[0], nil
	case 0:
		return nil, fmt.Errorf("no subnet %s", what)
	default:
		return nil, fmt.Errorf("%d subnets %s", len(found), what)
	}
}
