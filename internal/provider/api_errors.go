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
	"net/http"

	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

// apiStatus returns the HTTP status code of an API error returned by the SDK,
// or 0 if err is nil or not an API error.
func apiStatus(err error) int {
	var apiErr *verda.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}

// isNotFound reports whether err is an API 404 response.
func isNotFound(err error) bool { return apiStatus(err) == http.StatusNotFound }

// isConflict reports whether err is an API 409 response.
func isConflict(err error) bool { return apiStatus(err) == http.StatusConflict }
