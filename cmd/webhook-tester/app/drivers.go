package app

type storageDriverName string

const (
	storageDriverMemory storageDriverName = "memory"
	storageDriverRedis  storageDriverName = "redis"
	storageDriverFS     storageDriverName = "fs"
)

type pubSubDriverName string

const (
	pubSubDriverMemory pubSubDriverName = "memory"
	pubSubDriverRedis  pubSubDriverName = "redis"
)

type tunnelDriverName string

const tunnelDriverNgrok tunnelDriverName = "ngrok"
