// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package daemonset

import "sync"

// KeyLock provides a concurrency-safe key based lock
type KeyLock struct {
	locks   map[string]*sync.Mutex // map of locks, one sync.Mutex per key
	mapLock sync.Mutex             // to make the map safe concurrently
}

func NewKeyLock() *KeyLock {
	return &KeyLock{locks: make(map[string]*sync.Mutex)}
}

func (k *KeyLock) Lock(key string) {
	k.getLockBy(key).Lock()
}

func (k *KeyLock) Unlock(key string) {
	k.getLockBy(key).Unlock()
}

func (k *KeyLock) getLockBy(key string) *sync.Mutex {
	k.mapLock.Lock()
	defer k.mapLock.Unlock()

	ret, ok := k.locks[key]
	if ok {
		// if lock already exists in the map, just return it
		return ret
	}

	// otherwise, create and store a lock for the new `key`
	ret = &sync.Mutex{}
	k.locks[key] = ret
	return ret
}
