package hls

import (
	"io"
	"log"
	"math"
	"net/http"

	"github.com/avast/retry-go"
	mapset "github.com/deckarep/golang-set/v2"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type PrefetchPlaylist struct {
	playlistId    string
	playlistClips []string
	clipToIndex   cmap.ConcurrentMap[string, int]
	fetchedClips  cmap.ConcurrentMap[string, []byte]
}

func NewPrefetchPlaylist(playlistId string, playlistClips []string) *PrefetchPlaylist {
	clipToIndex := cmap.New[int]()
	fetchedClips := cmap.New[[]byte]()

	for index, clip := range playlistClips {
		clipToIndex.Set(clip, index)
	}
	return &PrefetchPlaylist{
		playlistId:    playlistId,
		playlistClips: playlistClips,
		clipToIndex:   clipToIndex,
		fetchedClips:  fetchedClips,
	}
}

func (m PrefetchPlaylist) getNextPrefetchClips(clipUrl string, count int) []string {

	clipIndex, ok := m.clipToIndex.Get(clipUrl)
	if !ok {
		return []string{}
	}
	lastCliPindex := math.Min(float64(clipIndex+count), float64(len(m.playlistClips)-1))

	return m.playlistClips[clipIndex+1 : int(lastCliPindex)]
}

func (m PrefetchPlaylist) addClip(clipUrl string, data []byte) {
	m.fetchedClips.Set(clipUrl, data)
}

type Prefetcher struct {
	clipPrefetchCount    int
	currentlyPrefetching mapset.Set[string]
	playlistInfo         map[string]*PrefetchPlaylist
}

func (p Prefetcher) GetFetchedClip(playlistId string, clipUrl string) ([]byte, bool) {
	playlist, ok := p.playlistInfo[playlistId]

	if !ok {

		return nil, false
	}

	data, foundClip := playlist.fetchedClips.Get(clipUrl)

	clipIndex, foundIndex := playlist.clipToIndex.Get(clipUrl)

	if foundIndex {

		firstClip := math.Min(float64(clipIndex+1), float64(len(playlist.playlistClips)-1))

		go p.prefetchClips(playlist.playlistClips[int(firstClip)], playlistId)
	}

	if !foundClip {
		return nil, false
	} else {
		return data, ok
	}

}

func (p Prefetcher) prefetchClips(clipUrl string, playlistId string) error {
	playlist, ok := p.playlistInfo[playlistId]
	if !ok {
		return nil
	}

	nextClips := playlist.getNextPrefetchClips(clipUrl, p.clipPrefetchCount)
	for _, clip := range nextClips {
		go func(clip string) {
			if p.currentlyPrefetching.Contains(clip) || playlist.fetchedClips.Has(clip) {
				return
			}
			p.currentlyPrefetching.Add(clip)

			data, err := fetchClip(clip)

			if err != nil {
				log.Printf("Error fetching clip %s: %v", clip, err)
				p.currentlyPrefetching.Remove(clip)
				return
			}
			log.Printf("Fetched clip %s", clip)
			p.currentlyPrefetching.Remove(clip)
			playlist.addClip(clip, data)
			return
		}(clip)

	}
	return nil
}

func fetchClip(clipUrl string) ([]byte, error) {
	var resp *http.Response
	err := retry.Do(
		func() error {
			var err error
			resp, err = http.Get(clipUrl)
			return err
		},
		retry.Attempts(3),
		retry.OnRetry(func(n uint, err error) {
			log.Printf("Retrying request after error: %v", err)
		}),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	// do something with the response
	return bytes, nil
}

func NewPrefetcher(clipPrefetchCount int) *Prefetcher {
	return &Prefetcher{
		clipPrefetchCount:    clipPrefetchCount,
		currentlyPrefetching: mapset.NewSet[string](),
		playlistInfo:         map[string]*PrefetchPlaylist{},
	}
}
