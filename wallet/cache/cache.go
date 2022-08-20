package cache

import (
	"errors"
	"github.com/TopiaNetwork/topia/wallet/key_store"
	"sync"
)

type AddrItemInCache struct {
	EncKeyItem []byte

	lock bool
}

var cacheMutex sync.RWMutex
var (
	addr_Cache   = make(map[string]AddrItemInCache)
	enable_Cache = true // default enabled
)

var (
	enableKeyNotSetErr = errors.New("enable key hasn't been set")
	addrNotExistErr    = errors.New("addr doesn't exist")
)

func GetAddrLock(addr string) (bool, error) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	_, find := addr_Cache[addr]
	if !find {
		return false, errors.New("didn't find the address: " + addr)

	}
	return addr_Cache[addr].lock, nil
}

func SetAddrLock(addr string, set bool) error {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	_, find := addr_Cache[addr]
	if !find {
		return errors.New("didn't find the address: " + addr)
	}

	addr_Cache[addr] = AddrItemInCache{
		EncKeyItem: addr_Cache[addr].EncKeyItem,
		lock:       set,
	}
	return nil
}

func SetAddrToCache(addr string, encData []byte) {
	cacheMutex.Lock()
	addr_Cache[addr] = AddrItemInCache{
		EncKeyItem: encData,
		lock:       false,
	}
	cacheMutex.Unlock()
}

func GetAddrFromCache(addr string) (AddrItemInCache, error) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	ret, ok := addr_Cache[addr]
	if !ok {
		return AddrItemInCache{}, addrNotExistErr
	}
	return ret, nil
}

func SetEnableToCache(set bool) {
	cacheMutex.Lock()
	enable_Cache = set
	cacheMutex.Unlock()
}

func GetEnableFromCache() bool {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	return enable_Cache
}

func GetKeysFromCache() (addrs []string) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	for k := range addr_Cache {
		addrs = append(addrs, k)
	}

	return addrs
}

func RemoveFromCache(key string) {
	cacheMutex.Lock()
	if key == key_store.EnableKey {
		enable_Cache = false
	} else {
		delete(addr_Cache, key)
	}
	cacheMutex.Unlock()
}
