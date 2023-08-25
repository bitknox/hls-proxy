package hls

import (
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/cristalhq/base64"
)

func ModifyM3u8(m3u8 string, host_url *url.URL) (string, error) {

	var re = regexp.MustCompile(`(?i)URI=["']([^"']+)["']`)
	var newManifest = strings.Builder{}
	host := os.Getenv("HOST")
	port := os.Getenv("PORT")

	parentPath := path.Dir(host_url.Path)
	host_url.Path = parentPath
	host_url.RawQuery = ""
	parentUrl := host_url.String()

	newManifest.Grow(len(m3u8))
	if strings.Contains(m3u8, "RESOLUTION=") {
		manifestAddr := "http://" + host + ":" + port + "/manifest?input="
		for _, line := range strings.Split(strings.TrimRight(m3u8, "\n"), "\n") {
			if len(line) == 0 {

				continue
			}
			if line[0] == '#' {
				//check for known tags and use regex to replace URI inside
				if strings.HasPrefix(line, "#EXT-X-MEDIA") {
					match := re.FindStringSubmatch(line)

					newManifest.WriteString(strings.Replace(line, match[1], "http://"+host+":"+port+"/manifest?input="+base64.StdEncoding.EncodeToString([]byte(parentUrl+"/"+match[1])), 1))
				} else {
					newManifest.WriteString(line)
				}
			} else if len(strings.TrimSpace(line)) > 0 {

				AddProxyUrl(manifestAddr, line, true, parentUrl, &newManifest)
			}
			newManifest.WriteString("\n")
		}
	} else {
		tsAddr := "http://" + host + ":" + port + "/"
		lines := strings.Split(strings.TrimRight(m3u8, "\n"), "\n")
		last_index := len(lines) - 1
		for i, line := range lines {

			if line[0] == '#' {
				newManifest.WriteString(line)
			} else {
				AddProxyUrl(tsAddr, line, false, parentUrl, &newManifest)
				newManifest.WriteString(".ts")
			}
			if i != last_index {
				newManifest.WriteString("\n")
			}
		}
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
