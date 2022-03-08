package netx

import (
	"context"
	"net/http"

	"github.com/projecteru2/yavirt/internal/errors"
)

// SimpleGet .
func SimpleGet(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return http.DefaultClient.Do(req.WithContext(ctx))
}
