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

func ModifyM3u8(m3u8 string, host_url *url.URL, prefetcher *Prefetcher) (string, error) {

	var newManifest = strings.Builder{}
	var host = model.Configuration.Host
	var port = model.Configuration.Port
	parentPath := path.Dir(host_url.Path)
	host_url.Path = parentPath
	host_url.RawQuery = ""
	host_url.Fragment = ""

	parentUrl := host_url.String()

	newManifest.Grow(len(m3u8))
	if strings.Contains(m3u8, "RESOLUTION=") {
		manifestAddr := "http://" + host + ":" + port + "/"
		for _, line := range strings.Split(strings.TrimRight(m3u8, "\n"), "\n") {
			if len(line) == 0 {

				continue
			}
			if line[0] == '#' {
				//check for known tags and use regex to replace URI inside
				if strings.HasPrefix(line, "#EXT-X-MEDIA") {
					match := re.FindStringSubmatch(line)
					newManifest.WriteString(strings.Replace(line, match[1], "http://"+host+":"+port+"/"+base64.StdEncoding.EncodeToString([]byte(parentUrl+"/"+match[1])), 1))
				} else {
					newManifest.WriteString(line)
				}
			} else if len(strings.TrimSpace(line)) > 0 {

				AddProxyUrl(manifestAddr, line, true, parentUrl, &newManifest)

			}
			newManifest.WriteString("\n")
		}
	} else {
		//most likely a master playlist containing the video elements

		var clipUrls []string
		var playlistId = counter.Add(1)
		var strId = strconv.Itoa(int(playlistId))

		tsAddr := "http://" + host + ":" + port + "/"
		for _, line := range strings.Split(strings.TrimRight(m3u8, "\n"), "\n") {

			if line[0] == '#' {
				newManifest.WriteString(line)
			} else {
				//the line here is a url to ts file that can be prefetched
				clipUrls = append(clipUrls, parentUrl+"/"+line)
				AddProxyUrl(tsAddr, line, false, parentUrl, &newManifest)
				newManifest.WriteString(".ts")
				newManifest.WriteString("?pId=" + strId)
			}
			newManifest.WriteString("\n")
		}

		prefetcher.AddPlaylistToCache(strId, clipUrls)

	}

	return newManifest.String(), nil
}

func AddProxyUrl(baseAddr string, url string, isManifest bool, parentUrl string, builder *strings.Builder) {
	builder.WriteString(baseAddr)
	if strings.HasPrefix(url, "http") {
		builder.WriteString(base64.StdEncoding.EncodeToString([]byte(url)))
	} else {
		builder.WriteString(base64.StdEncoding.EncodeToString([]byte(parentUrl + "/" + url)))
	}
}
