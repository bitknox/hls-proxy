package hls

import (
	"io"
	"log"
	"math"
	"net/http"
	"runtime"
	"time"

	http_retry "github.com/bitknox/hls-proxy/http_retry"
	mapset "github.com/deckarep/golang-set/v2"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type Cleanable interface {
	setJanitor(j *Janitor)
	getJanitor() *Janitor
	Clean()
}

type CacheItem[T any] struct {
	Data       T
	Expiration time.Time
}

type Janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *Janitor) Run(c Cleanable) {
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			c.Clean()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func runJanitor(c Cleanable, ci time.Duration) {
	j := &Janitor{
		Interval: ci,
		stop:     make(chan bool),
	}
	c.setJanitor(j)
	go j.Run(c)
}

type PrefetchPlaylist struct {
	clipRetention time.Duration
	janitor       *Janitor
	playlistId    string
	playlistClips []string
	clipToIndex   cmap.ConcurrentMap[string, int]
	fetchedClips  cmap.ConcurrentMap[string, CacheItem[[]byte]]
}

func newPrefetchPlaylist(playlistId string, playlistClips []string, clipRetention time.Duration) *PrefetchPlaylist {
	clipToIndex := cmap.New[int]()
	fetchedClips := cmap.New[CacheItem[[]byte]]()

	for index, clip := range playlistClips {
		clipToIndex.Set(clip, index)
	}
	return &PrefetchPlaylist{
		playlistId:    playlistId,
		playlistClips: playlistClips,
		clipToIndex:   clipToIndex,
		fetchedClips:  fetchedClips,
		clipRetention: clipRetention,
	}
}

func (m PrefetchPlaylist) getJanitor() *Janitor {
	return m.janitor
}

func newPrefetchPlaylistWithJanitor(playlistId string, playlistClips []string, janitorInterval time.Duration, clipRetention time.Duration) *PrefetchPlaylist {
	p := newPrefetchPlaylist(playlistId, playlistClips, clipRetention)
	initJanitor(p, janitorInterval)
	return p
}

func initJanitor(cache Cleanable, ci time.Duration) {
	if ci <= 0 {
		return
	}
	runtime.SetFinalizer(cache, func(cache Cleanable) {
		stopJanitor(cache.getJanitor())
	})
	runJanitor(cache, ci)
}

func stopJanitor(j *Janitor) {
	j.stop <- true
}

func (m PrefetchPlaylist) setJanitor(j *Janitor) {
	m.janitor = j
}

func (m PrefetchPlaylist) Clean() {
	println("Cleaning playlist")
	currentTime := time.Now()
	for clipUrl, clipItem := range m.fetchedClips.Items() {
		if clipItem.Expiration.Before(currentTime) {
			println("Removing clip")
			m.fetchedClips.Remove(clipUrl)
		}
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
	expires := time.Now().Add(m.clipRetention)
	m.fetchedClips.Set(clipUrl, CacheItem[[]byte]{
		Data:       data,
		Expiration: expires,
	})
}

type Prefetcher struct {
	janitor              *Janitor
	clipPrefetchCount    int
	currentlyPrefetching mapset.Set[string]
	playlistInfo         cmap.ConcurrentMap[string, CacheItem[*PrefetchPlaylist]]
	playlistRetention    time.Duration
	clipRetention        time.Duration
}

func (p Prefetcher) GetFetchedClip(playlistId string, clipUrl string) ([]byte, bool) {
	playlistItem, ok := p.playlistInfo.Get(playlistId)

	if !ok {

		return nil, false
	}

	playlist := playlistItem.Data

	data, foundClip := playlist.fetchedClips.Get(clipUrl)

	clipIndex, foundIndex := playlist.clipToIndex.Get(clipUrl)

	if foundIndex {

		firstClip := math.Min(float64(clipIndex+1), float64(len(playlist.playlistClips)-1))

		go p.prefetchClips(playlist.playlistClips[int(firstClip)], playlistId)
	}

	if !foundClip {
		return nil, false
	} else {
		return data.Data, ok
	}

}

func (p Prefetcher) AddPlaylistToCache(playlistId string, clipUrls []string) {
	expires := time.Now().Add(p.playlistRetention)
	p.playlistInfo.Set(playlistId, CacheItem[*PrefetchPlaylist]{
		Data:       newPrefetchPlaylistWithJanitor(playlistId, clipUrls, 20*time.Second, p.playlistRetention),
		Expiration: expires,
	})
}

func (p Prefetcher) prefetchClips(clipUrl string, playlistId string) error {
	playlistItem, ok := p.playlistInfo.Get(playlistId)
	if !ok {
		return nil
	}

	playlist := playlistItem.Data

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
	request, err := http.NewRequest("GET", clipUrl, nil)

	resp, err := http_retry.ExecuteRetryableRequest(request)
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	// do something with the response
	return bytes, nil
}

func NewPrefetcher(clipPrefetchCount int, playlistRetention time.Duration, clipRetention time.Duration) *Prefetcher {
	return &Prefetcher{
		clipPrefetchCount:    clipPrefetchCount,
		currentlyPrefetching: mapset.NewSet[string](),
		playlistInfo:         cmap.New[CacheItem[*PrefetchPlaylist]](),
		playlistRetention:    playlistRetention,
		clipRetention:        clipRetention,
	}
}

func (p Prefetcher) setJanitor(j *Janitor) {
	p.janitor = j
}

func (p Prefetcher) getJanitor() *Janitor {
	return p.janitor
}

func (p Prefetcher) Clean() {
	println("Cleaning")
	currentTime := time.Now()
	for playlistId, playlistItem := range p.playlistInfo.Items() {
		if playlistItem.Expiration.Before(currentTime) {
			p.playlistInfo.Remove(playlistId)
		}
	}
}

func NewPrefetcherWithJanitor(clipPrefetchCount int, janitorInterval time.Duration, playlistRetention time.Duration, clipRetention time.Duration) *Prefetcher {
	p := NewPrefetcher(clipPrefetchCount, playlistRetention, clipRetention)
	initJanitor(p, janitorInterval)
	return p
}
