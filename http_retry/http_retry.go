package http_retry

import (
	"log"
	"net/http"

	"github.com/avast/retry-go"
)

func ExecuteRetryableRequest(request *http.Request, attempts int) (*http.Response, error) {
	var resp *http.Response
	err := retry.Do(
		func() error {
			var err error
			resp, err = http.DefaultClient.Do(request)
			return err
		},
		retry.Attempts(3),
		retry.OnRetry(func(n uint, err error) {
			log.Printf("Retrying request after error: %v", err)
		}),
	)
	if err != nil {
		return nil, err
	}

	return resp, nil

}
