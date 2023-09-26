package http_retry

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// DefaultHttpClient is the default http client used by the proxy
// We set a timeout of 15 seconds for the dialer to make sure requests don't hang and block the proxy
var DefaultHttpClient = http.Client{
	Timeout: 60 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Dial: (&net.Dialer{
			Timeout: 15 * time.Second,
		}).Dial,
	},
}
