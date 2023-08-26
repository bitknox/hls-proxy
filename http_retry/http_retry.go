package http_retry

import (
	"errors"
	"net/http"
	"time"

	"github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

func ExecuteRetryableRequest(request *http.Request, attempts int) (*http.Response, error) {
	var resp *http.Response
	err := retry.Do(
		func() error {
			var err error
			resp, err = http.DefaultClient.Do(request)
			if resp.StatusCode >= 300 || resp.StatusCode < 200 {
				return errors.New("Non 2xx status code")
			}

			return err
		},
		retry.Attempts(uint(attempts)),
		retry.Delay(time.Second),
		retry.OnRetry(func(n uint, err error) {
			log.Error("Retrying request after error:", err, n)
		}),
	)
	if err != nil {
		return nil, err
	}

	return resp, nil

}
