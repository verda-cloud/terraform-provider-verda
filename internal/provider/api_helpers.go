package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

func doVerdaRequest(ctx context.Context, client *verda.Client, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(encoded)
	}

	if result == nil {
		result = &struct{}{}
	}

	req, err := client.NewRequest(ctx, method, path, bodyReader)
	if err != nil {
		return err
	}

	_, err = client.Do(req, result)
	return err
}
