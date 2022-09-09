package cache

import (
	"errors"
	"github.com/TopiaNetwork/topia/wallet/key_store"
	lru "github.com/hashicorp/golang-lru"
	"sync"
)

type AddrItemInCache struct {
	EncKeyItem []byte

	lock bool
}

const addrCacheSize = 512 // hold 512 item max
var addrCache, _ = lru.New(addrCacheSize)

var (
	enableCacheMutex      sync.RWMutex
	enable_Cache          = true // default enabled
	defaultAddrCacheMutex sync.RWMutex
	defaultAddr_Cache     = "" // default void-string
)

var (
	addrNotExistErr = errors.New("addr doesn't exist in cache")
)

func GetAddrLock(addr string) (bool, error) {
	value, ok := addrCache.Get(addr)
	if !ok {
		return false, addrNotExistErr
	}
	item, ok := value.(AddrItemInCache)
	if !ok {
		return false, errors.New("invalid addr item type")
	}
	return item.lock, nil
}

func SetAddrLock(addr string, set bool) error {
	value, ok := addrCache.Get(addr)
	if !ok {
		return addrNotExistErr
	}
	item, ok := value.(AddrItemInCache)
	if !ok {
		return errors.New("invalid addr item type")
	}
	item.lock = set
	addrCache.Add(addr, item)

	return nil
}

func SetAddrToCache(addr string, encData []byte) {
	item := AddrItemInCache{
		EncKeyItem: encData,
		lock:       false,
	}
	addrCache.Add(addr, item)
}

func GetAddrFromCache(addr string) (AddrItemInCache, error) {
	value, ok := addrCache.Get(addr)
	if !ok {
		return AddrItemInCache{}, addrNotExistErr
	}
	item, ok := value.(AddrItemInCache)
	if !ok {
		return AddrItemInCache{}, errors.New("invalid addr item type")
	}
	return item, nil
}

func SetEnableToCache(set bool) {
	enableCacheMutex.Lock()
	enable_Cache = set
	enableCacheMutex.Unlock()
}

func GetEnableFromCache() (ret bool) {
	enableCacheMutex.RLock()
	ret = enable_Cache
	enableCacheMutex.RUnlock()

	return ret
}

func SetDefaultAddrToCache(defaultAddr string) {
	defaultAddrCacheMutex.Lock()
	defaultAddr_Cache = defaultAddr
	defaultAddrCacheMutex.Unlock()
}

func GetDefaultAddrFromCache() (defaultAddr string) {
	defaultAddrCacheMutex.RLock()
	defaultAddr = defaultAddr_Cache
	defaultAddrCacheMutex.RUnlock()

	return defaultAddr
}

func GetKeysFromCache() (addrs []string) {
	keys := addrCache.Keys()
	addrs = make([]string, len(keys))
	for i := range keys {
		addrs[i] = keys[i].(string)
	}

	return addrs
}

func RemoveFromCache(key string) {
	if key == key_store.EnableKey {
		enableCacheMutex.Lock()
		enable_Cache = false
		enableCacheMutex.Unlock()
	} else if key == key_store.DefaultAddrKey {
		defaultAddrCacheMutex.Lock()
		defaultAddr_Cache = ""
		defaultAddrCacheMutex.Unlock()
	} else {
		addrCache.Remove(key)
	}
}
