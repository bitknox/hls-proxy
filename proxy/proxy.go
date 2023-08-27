package proxy

import (
	"net"
	"net/http"
	"time"
)

// base useragent string
const USER_AGENT string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "

// DefaultHttpClient is the default http client used by the proxy
// We set a timeout of 15 seconds for the dialer to make sure requests don't hang and block the proxy
var DefaultHttpClient = http.Client{
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 15 * time.Second,
		}).Dial,
	},
}
