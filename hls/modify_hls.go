package hls

import (
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/bitknox/hls-proxy/model"
	"github.com/cristalhq/base64"
)

var counter atomic.Int32
var re = regexp.MustCompile(`(?i)URI=["']([^"']+)["']`)

/*
*	Very barebones m3u8 parser that will replace the URI inside the manifest with a proxy url
*	It only supports a subset of the m3u8 tags and will not work with all m3u8 files
*   It should probably be replaced with a proper m3u8 parser
 */

func ModifyM3u8(m3u8 string, host_url *url.URL, prefetcher *Prefetcher, input *model.Input) (string, error) {

	var newManifest = strings.Builder{}
	var host = model.Configuration.Host

	parentPath := path.Dir(host_url.Path)
	host_url.Path = parentPath
	host_url.RawQuery = ""
	host_url.Fragment = ""

	parentUrl := host_url.String()

	masterProxyUrl := ""
	//if user wants https, we should use it
	if model.Configuration.UseHttps {
		masterProxyUrl = "https://" + host + "/"
	} else {
		masterProxyUrl = "http://" + host + "/"
	}

	newManifest.Grow(len(m3u8))
	if strings.Contains(m3u8, "RESOLUTION=") {
		manifestAddr := masterProxyUrl
		for _, line := range strings.Split(strings.TrimRight(m3u8, "\n"), "\n") {
			if len(line) == 0 {

				continue
			}
			if line[0] == '#' {
				//check for known tags and use regex to replace URI inside
				if strings.HasPrefix(line, "#EXT-X-MEDIA") {

					handleUriTag(line, parentUrl, input, &newManifest, masterProxyUrl)
				} else {
					newManifest.WriteString(line)
				}
			} else if len(strings.TrimSpace(line)) > 0 {

				AddProxyUrl(manifestAddr, line, true, parentUrl, &newManifest, input)

			}
			newManifest.WriteString("\n")
		}
	} else {
		//most likely a master playlist containing the video elements

		var clipUrls []string
		var playlistId = counter.Add(1)
		var strId = strconv.Itoa(int(playlistId))

		tsAddr := masterProxyUrl
		for _, line := range strings.Split(strings.TrimRight(m3u8, "\n"), "\n") {

			if line[0] == '#' {
				//check for key and replace URI with proxy url
				if strings.HasPrefix(line, "#EXT-X-KEY") {
					handleUriTag(line, parentUrl, input, &newManifest, masterProxyUrl)
				} else {
					newManifest.WriteString(line)
				}
			} else {
				//the line here is a url to ts file that can be prefetched
				clipUrls = append(clipUrls, parentUrl+"/"+line)
				AddProxyUrl(tsAddr, line, false, parentUrl, &newManifest, input)
				newManifest.WriteString(".ts")
				newManifest.WriteString("?pId=" + strId)
			}
			newManifest.WriteString("\n")
		}

		prefetcher.AddPlaylistToCache(strId, clipUrls)

	}

	return newManifest.String(), nil
}

func handleUriTag(line string, parentUrl string, input *model.Input, newManifest *strings.Builder, masterProxyUrl string) {

	match := re.FindStringSubmatch(line)
	proxyUrl := ""
	if strings.HasPrefix(match[1], "http") || strings.HasPrefix(match[1], "https") {
		proxyUrl = match[1]
	} else {
		proxyUrl = parentUrl + "/" + match[1]
	}

	if input.Referer != "" {
		proxyUrl += "|" + input.Referer
	}
	if input.Origin != "" {
		proxyUrl += "|" + input.Origin
	}
	encodedProxyUrl := base64.StdEncoding.EncodeToString([]byte(proxyUrl))
	newManifest.WriteString(strings.Replace(line, match[1], masterProxyUrl+encodedProxyUrl, 1))
}

func AddProxyUrl(baseAddr string, url string, isManifest bool, parentUrl string, builder *strings.Builder, input *model.Input) {

	proxyUrl := url
	if input.Referer != "" {
		proxyUrl += "|" + input.Referer
	}
	if input.Origin != "" {
		proxyUrl += "|" + input.Origin
	}
	builder.WriteString(baseAddr)
	if strings.HasPrefix(url, "http") || strings.HasPrefix(url, "https") {
		builder.WriteString(base64.StdEncoding.EncodeToString([]byte(proxyUrl)))
	} else {
		builder.WriteString(base64.StdEncoding.EncodeToString([]byte(parentUrl + "/" + proxyUrl)))
	}
}
