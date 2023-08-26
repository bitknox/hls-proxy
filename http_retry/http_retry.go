package http_retry

import (
	"net/http"

	"github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
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
			log.Error("Retrying request after error:", err, n)
		}),
	)
	if err != nil {
		return nil, err
	}

	return resp, nil

}
