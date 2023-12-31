package hls

import (
	"math"
	"net/http"
	"runtime"
	"time"

	http_retry "github.com/bitknox/hls-proxy/http_retry"
	"github.com/bitknox/hls-proxy/model"
	mapset "github.com/deckarep/golang-set/v2"
	cmap "github.com/orcaman/concurrent-map/v2"
	log "github.com/sirupsen/logrus"
)

// Interface for structures that can be cleaned by a janitor
type Cleanable interface {
	setJanitor(j *Janitor)
	getJanitor() *Janitor
	Clean()
}

// CacheItem is a struct that holds the data and expiration time of a cached item
// This makes it possible for the janitor to clean the cache
type CacheItem[T any] struct {
	Data       T
	Expiration time.Time
}

// Janitor is a struct that holds the interval and stop channel for a janitor
type Janitor struct {
	Interval time.Duration
	stop     chan bool
}

// Run is a function that runs the janitor, and cleans the cache at the specified interval
// It stops when the stop recives a value
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

// Structure that holds the playlist clips and the cached clips
type PrefetchPlaylist struct {
	clipRetention time.Duration
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

func (m PrefetchPlaylist) Clean() {
	log.Debug("Cleaning playlist ", m.playlistId)
	currentTime := time.Now()
	for clipUrl, clipItem := range m.fetchedClips.Items() {
		if clipItem.Expiration.Before(currentTime) {
			log.Debug("Removed clip from ", m.playlistId, " with url", clipUrl)
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
	firstclipIndex := clipIndex + 1
	if firstclipIndex > int(lastCliPindex) {
		return []string{}
	}
	return m.playlistClips[firstclipIndex:int(lastCliPindex)]
}

func (m PrefetchPlaylist) addClip(clipUrl string, data []byte) {
	expires := time.Now().Add(m.clipRetention)
	m.fetchedClips.Set(clipUrl, CacheItem[[]byte]{
		Data:       data,
		Expiration: expires,
	})
}

/*
Structure that handles prefetching of clips, and caching of playlists
Supports automatic cleanup of cached playlists and clips if using the NewPrefetcherWithJanitor constructor
*/
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
	log.Debug("Adding playlist to cache ", playlistId)
	expires := time.Now().Add(p.playlistRetention)
	p.playlistInfo.Set(playlistId, CacheItem[*PrefetchPlaylist]{
		Data:       newPrefetchPlaylist(playlistId, clipUrls, p.clipRetention),
		Expiration: expires,
	})
}

// We might want to introduce a delay between requests to the same host
func (p Prefetcher) prefetchClips(clipUrl string, playlistId string) error {
	playlistItem, ok := p.playlistInfo.Get(playlistId)
	if !ok {
		return nil
	}

	playlist := playlistItem.Data

	nextClips := playlist.getNextPrefetchClips(clipUrl, p.clipPrefetchCount)

	throttle := time.NewTicker(time.Second / time.Duration(model.Configuration.Throttle))
	defer throttle.Stop()
	for _, clip := range nextClips {
		//if we are already in the process of fetching the clip, or we already have it cached, skip it

		if p.currentlyPrefetching.Contains(clip) || playlist.fetchedClips.Has(clip) {

			continue
		}
		p.currentlyPrefetching.Add(clip)

		<-throttle.C

		go func(clip string) {

			data, err := fetchClip(clip)

			if err != nil {
				log.Debug("Error fetching clip ", clip, err)
				p.currentlyPrefetching.Remove(clip)
				return
			}
			log.Debug("Fetched clip ", clip)
			p.currentlyPrefetching.Remove(clip)
			playlist.addClip(clip, data)
			log.Debug("Number of cached clips", playlist.fetchedClips.Count())

			return
		}(clip)

	}

	return nil
}

func fetchClip(clipUrl string) ([]byte, error) {
	request, err := http.NewRequest("GET", clipUrl, nil)

	resp, err := http_retry.ExecuteRetryClipRequest(request, model.Configuration.Attempts)

	if err != nil {
		log.Error("Error fetching clip ", clipUrl, err)
		return nil, err
	}
	// do something with the response
	return resp, nil
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
	log.Debug("Cleaning playlist cache")
	currentTime := time.Now()
	for playlistId, playlistItem := range p.playlistInfo.Items() {
		if playlistItem.Expiration.Before(currentTime) {
			log.Debug("Removed playlist ", playlistId)
			p.playlistInfo.Remove(playlistId)
		} else {
			playlist := playlistItem.Data
			playlist.Clean()
		}
	}
}

func NewPrefetcherWithJanitor(clipPrefetchCount int, janitorInterval time.Duration, playlistRetention time.Duration, clipRetention time.Duration) *Prefetcher {
	p := NewPrefetcher(clipPrefetchCount, playlistRetention, clipRetention)
	initJanitor(p, janitorInterval)
	return p
}
