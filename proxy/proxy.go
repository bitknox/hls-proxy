package proxy

import (
	"net"
	"net/http"
	"time"
)

// base useragent string
const USER_AGENT string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "

var DefaultHttpClient = http.Client{
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 15 * time.Second,
		}).Dial,
	},
}
