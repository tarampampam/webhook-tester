package storage_test

import (
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/internal/storage"
)

func TestInMemory_Session_CreateReadDelete(t *testing.T) {
	testSessionCreateReadDelete(t, func(sTTL time.Duration, maxReq uint32) storageToTest {
		return storage.NewInMemory(sTTL, maxReq)
	})
}

func TestInMemory_Request_CreateReadDelete(t *testing.T) {
	testRequestCreateReadDelete(t, func(sTTL time.Duration, maxReq uint32) storageToTest {
		return storage.NewInMemory(sTTL, maxReq)
	})
}

func TestInMemory_RaceProvocation(t *testing.T) {
	testRaceProvocation(t, func(sTTL time.Duration, maxReq uint32) storageToTest {
		return storage.NewInMemory(sTTL, maxReq)
	})
}
