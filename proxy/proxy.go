package proxy

import (
	"net/http"
)

// base useragent string
const USER_AGENT string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "

// Proxy is a struct that holds the http client
type ProxyClient struct {
	Client *http.Client
}

// a global proxy instance
var Proxy *ProxyClient = &ProxyClient{
	Client: &http.Client{},
}
