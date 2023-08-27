# Go hls-proxy 📺

A simple proxy server that parses m3u8 manifets and proxies all requests. This is useful for adding headers, prefetching clips or other custom logic when loading streams that cannot be modified directly at the source.


## Getting Started ▶️

### Dependencies

* [golang](https://go.dev/doc/install)

### Installing 👨‍💻

```bash
git clone https://github.com/bitknox/hls-proxy.git
cd hls-proxy
go install
hls-proxy
```


## Help 🆘

```bash
hls-proxy h
```

#### Overview of options

```bash
--prefetch                  prefetch ts files (default: true)
--segments value            how many segments to prefetch (default: 30)
--throttle value            how much to throttle prefetch requests (requests per second) (default: 5)
--janitor-interval value    how often should the janitor clean the cache (default: 20s)
--attempts value            how many times to retry a request for a ts file (default: 3)
--clip-retention value      how long to keep ts files in cache (default: 30m0s)
--playlist-retention value  how long to keep playlists in cache (default: 5h0m0s)
--host value                hostname to attach to proxy url
--port value                port to attach to proxy url (default: 1323)
--log-level value           log level (default: "PRODUCTION")
--help, -h                  show help
```

## Contributing 🧑‍🏭

Contributions are always welcome. This is one of my first projets in golang, so I'm sure there room for a lot of improvement.

## Authors 📗

[@bitknox](https://github.com/bitknox)

## Version History 🗎

* 1.0
    * Initial Release
    * See [commit change]() or See [release history]()

## License ©️

This project is licensed under the MIT License - see the LICENSE file for details

## Acknowledgments 🤚

Inspiration:
* [HLS-Proxy](https://github.com/warren-bank/HLS-Proxy) by [@warren-bank](https://github.com/warren-bank)
