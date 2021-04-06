package config

import "time"

type pubSubDriver string

const (
	PubSubDriverMemory pubSubDriver = "memory"
	PubSubDriverRedis  pubSubDriver = "redis"
)

func (d pubSubDriver) String() string { return string(d) }

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
	MaxRequestBodySize   uint32 // maximal webhook request body size (in bytes), zero means unlimited

	StorageDriver storageDriver
	PubSubDriver  pubSubDriver

	WebSockets struct {
		MaxClients  uint32        // zero means unlimited
		MaxLifetime time.Duration // zero means unlimited
	}
}
