package netx

import (
	"context"
	"net/http"

	"github.com/cockroachdb/errors"
)

// SimpleGet .
func SimpleGet(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return http.DefaultClient.Do(req.WithContext(ctx))
}
