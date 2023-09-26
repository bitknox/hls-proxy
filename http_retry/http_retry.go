package http_retry

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

/*
 * The two functions in here differ in that one returns the response body as a byte array,
 * and the other returns the response object that has to be handeled by the caller.
 */

func ExecuteRetryableRequest(request *http.Request, attempts int) (*http.Response, error) {
	request.Close = true
	var resp *http.Response
	err := retry.Do(
		func() error {
			var err error
			resp, err = DefaultHttpClient.Do(request)
			if(resp != nil) {
			if resp.StatusCode >= 300 || resp.StatusCode < 200 {
				return errors.New("Non 2xx status code")
			}}

			return err
		},
		retry.Attempts(uint(attempts)),
		retry.Delay(time.Second* 3),
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
			resp, err := DefaultHttpClient.Do(request)

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
		retry.Delay(time.Second * 2),
		retry.OnRetry(func(n uint, err error) {
			log.Error("Retrying request after error:", err, n)
		}),
	)
	if err != nil {
		return nil, err
	}

	return responseBytes, nil

}
