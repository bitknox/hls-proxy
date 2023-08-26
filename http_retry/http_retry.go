package http_retry

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/avast/retry-go"
	"github.com/bitknox/hls-proxy/proxy"
	log "github.com/sirupsen/logrus"
)

func ExecuteRetryableRequest(request *http.Request, attempts int) (*http.Response, error) {
	request.Close = true
	var resp *http.Response
	err := retry.Do(
		func() error {
			var err error
			resp, err = proxy.DefaultHttpClient.Do(request)
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

func ExecuteRetryClipRequest(request *http.Request, attempts int) ([]byte, error) {
	request.Close = true
	var responseBytes []byte
	err := retry.Do(
		func() error {
			resp, err := proxy.DefaultHttpClient.Do(request)

			if err != nil {
				return err
			}

			if resp.StatusCode >= 300 || resp.StatusCode < 200 {
				return errors.New("Non 2xx status code")
			}
			defer resp.Body.Close()
			bytes, err := io.ReadAll(resp.Body)

			if err != nil {
				return err
			}
			responseBytes = bytes

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

	return responseBytes, nil

}
