package http_retry

import (
	"net"
	"net/http"
	"time"
)

// DefaultHttpClient is the default http client used by the proxy
// We set a timeout of 15 seconds for the dialer to make sure requests don't hang and block the proxy
var DefaultHttpClient = http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 15 * time.Second,
		}).Dial,
	},
}
