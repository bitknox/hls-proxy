package limiter

import (
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
)

type BasicLimiter struct {
	limit       uint
	limiterType LimiterMode
	buckets     cmap.ConcurrentMap[string, chan bool]
}

func NewBasicLimiter(limit uint, limiterType LimiterMode) Limiter {
	return &BasicLimiter{
		limit:       limit,
		limiterType: limiterType,
		buckets:     cmap.New[chan bool](),
	}
}

func (l *BasicLimiter) Wait(pId string) {
	channel, ok := l.buckets.Get(pId)

	if !ok {

		switch l.limiterType {
		case Concurrent:
			channel = maxConcurrentRequestsLimiter(l.limit)
		case PerSecond:
			channel = maxRequestsPerSecondLimiter(l.limit)
		}
		l.buckets.Set(pId, channel)
		channel <- true
		return
	}

	channel <- true

}

func (l *BasicLimiter) Release(pId string) {
	switch l.limiterType {
	case Concurrent:
		channel, ok := l.buckets.Get(pId)
		if !ok {
			panic("Release called without Wait")
		}
		<-channel

	case PerSecond:
		// Do nothing
	}
}

// this limits requests to x per second, this could be changed to a sliding window approach
// to have a more even distribution of requests
// instead of a burst of requests every second
func maxRequestsPerSecondLimiter(requestsPerSecond uint) chan bool {

	maxRequestsPerSecondLimiterChannel := make(chan bool, requestsPerSecond)

	// 1. If channel is fully available -> for loop will block
	// 2. If channnel has at least one value -> allow another request after a second
	// 		for desired amount of requests per second
	for i := uint(0); i < requestsPerSecond; i++ {
		go func() {
			for {
				<-time.After(time.Second)
				<-maxRequestsPerSecondLimiterChannel
			}
		}()
	}

	return maxRequestsPerSecondLimiterChannel
}

// Limiter that limits the amount of concurrent requests
func maxConcurrentRequestsLimiter(concurrentRequests uint) chan bool {
	return make(chan bool, concurrentRequests)
}
