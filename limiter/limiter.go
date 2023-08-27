package limiter

type LimiterMode string

type Limiter interface {
	Wait(pId string)
	Release(pId string)
}

const (
	Concurrent LimiterMode = "concurrent"
	PerSecond              = "PerSecond"
)
