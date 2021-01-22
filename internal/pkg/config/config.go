package config

import "time"

type broadcastDriver string

const (
	BroadcastDriverNone   broadcastDriver = "none"
	BroadcastDriverPusher broadcastDriver = "pusher"
)

func (d broadcastDriver) String() string { return string(d) }

type storageDriver string

const (
	StorageDriverMemory storageDriver = "memory"
	StorageDriverRedis  storageDriver = "redis"
)

func (d storageDriver) String() string { return string(d) }

type Config struct {
	MaxRequests          uint16
	SessionTTL           time.Duration
	IgnoreHeaderPrefixes []string

	StorageDriver   storageDriver
	BroadcastDriver broadcastDriver

	Pusher struct {
		AppID   string
		Key     string
		Secret  string
		Cluster string
	}
}
