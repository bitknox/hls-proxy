package hls

import (
	"io"
	"net/http"
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
		var decryptionKey string
		var initialVector int
		var strId = strconv.Itoa(int(playlistId))

		tsAddr := masterProxyUrl
		for _, line := range strings.Split(strings.TrimRight(m3u8, "\n"), "\n") {

			if line[0] == '#' {
				//Check for the media sequence tag and set the sequence number, this is used as the
				// initialisation vector for the decryption for each segment
				if strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE") {
					//get the sequence number
					sequenceNumber, err := strconv.Atoi(strings.Split(line, ":")[1])
					if err != nil {
						return "", err
					}
					//set the sequence number
					initialVector = sequenceNumber
				}

				//check for key and replace URI with proxy url

				if strings.HasPrefix(line, "#EXT-X-KEY") && model.Configuration.DecryptSegments {

					//Get the uri and fetch the decryption key
					_, proxyUrl := getUrlForEmbeddedEntry(line, parentUrl)
					resp, err := http.Get(proxyUrl)
					if err != nil {
						return "", err
					}
					defer resp.Body.Close()
					body, err := io.ReadAll(resp.Body)

					if err != nil {
						return "", err
					}
					//convert the key to base64
					decryptionKey = base64.URLEncoding.EncodeToString(body)

				} else if strings.HasPrefix(line, "#EXT-X-KEY") {
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
				if decryptionKey != "" {
					newManifest.WriteString("&key=" + decryptionKey)
					newManifest.WriteString("&iv=" + strconv.Itoa(initialVector))
					initialVector++
				}
			}
			newManifest.WriteString("\n")
		}

		prefetcher.AddPlaylistToCache(strId, clipUrls)

	}

	return newManifest.String(), nil
}

func handleUriTag(line string, parentUrl string, input *model.Input, newManifest *strings.Builder, masterProxyUrl string) {

	original, proxyUrl := getUrlForEmbeddedEntry(line, parentUrl)

	if input.Referer != "" {
		proxyUrl += "|" + input.Referer
	}
	if input.Origin != "" {
		proxyUrl += "|" + input.Origin
	}
	encodedProxyUrl := base64.StdEncoding.EncodeToString([]byte(proxyUrl))
	newManifest.WriteString(strings.Replace(line, original, masterProxyUrl+encodedProxyUrl, 1))
}

func getUrlForEmbeddedEntry(url string, parentUrl string) (string, string) {
	match := re.FindStringSubmatch(url)
	if strings.HasPrefix(match[1], "http") || strings.HasPrefix(match[1], "https") {
		return match[1], match[1]
	} else {
		return match[1], parentUrl + "/" + match[1]
	}
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
