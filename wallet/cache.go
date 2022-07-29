package wallet

import (
	"errors"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	"sync"
	"time"
)

type keyStoreBackend interface {
	getAddrItemFromBackend(addr string) (keyItem, error)
	getEnableFromBackend() (bool, error)
	listAddrsFromBackend() (addrs []string, err error)
}

type addrItemInCache struct {
	keyItem

	lock bool
}

var cacheMutex sync.RWMutex // protect whole cache
var (
	addr_Cache        = make(map[string]addrItemInCache)
	enable_Cache      = true // default enabled
	defaultAddr_Cache = ""   // default void-string
)

var once sync.Once

func syncCacheRoutine(nSeconds time.Duration, store keyStore, logger tplog.Logger) {

	for {
		err := loadKeysToCache(store)
		if err != nil {
			logger.Errorf("synchronize cache err: ", err)
		}

		time.Sleep(nSeconds * time.Second)
	}

}

func runSyncCacheRoutineOnce(nSeconds time.Duration, store keyStore, logger tplog.Logger) {
	once.Do(func() {
		go syncCacheRoutine(nSeconds, store, logger)
	})
}

// loadKeysToCache is called when after keyStore implementation's initialization and keys' files get synchronized periodically.
func loadKeysToCache(store keyStore) error {
	var ksb keyStoreBackend

	if fks, ok := store.(*fileKeyStore); ok {
		ksb = fks
	} else if kri, ok := store.(*keyringImp); ok {
		ksb = kri
	} else {
		return errors.New("invalid keyStore implementation")
	}

	addrs, err := ksb.listAddrsFromBackend()
	if err != nil {
		return err
	}

	for _, addr := range addrs { // synchronize backend addrs to addr_Cache
		if _, find := addr_Cache[addr]; !find {
			item, err := ksb.getAddrItemFromBackend(addr)
			if err != nil {
				return err
			}

			cacheMutex.Lock()
			addr_Cache[addr] = addrItemInCache{
				keyItem: item,
				lock:    false, // default unlock
			}
			cacheMutex.Unlock()
		}
	}

	cacheMutex.Lock()
	if len(addr_Cache) != len(addrs) { // Addr exists in addr_Cache but not in backend(eg: User deleted keyFile directly instead of via API)
		for k := range addr_Cache {
			match := false
			for _, fileKey := range addrs {
				if k == fileKey {
					match = true
					break
				}
			}
			if !match {
				delete(addr_Cache, k)
			}
		}
	}
	cacheMutex.Unlock()

	enable, err := ksb.getEnableFromBackend()
	if err != nil {
		cacheMutex.Lock()
		enable_Cache = false
		cacheMutex.Unlock()
		return err
	} else {
		cacheMutex.Lock()
		enable_Cache = enable
		cacheMutex.Unlock()

	}

	defaultAddr, err := getDefaultAddrFromBackend()
	cacheMutex.Lock()
	if err != nil {
		defaultAddr_Cache = ""
		return err
	} else {
		defaultAddr_Cache = defaultAddr
	}
	cacheMutex.Unlock()
	return nil
}

func getAddrLock(addr string) (bool, error) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	_, find := addr_Cache[addr]
	if !find {
		return false, errors.New("didn't find the address: " + addr)

	}
	return addr_Cache[addr].lock, nil
}

func setAddrLock(addr string, set bool) error {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	_, find := addr_Cache[addr]
	if !find {
		return errors.New("didn't find the address: " + addr)
	}

	addr_Cache[addr] = addrItemInCache{
		keyItem: addr_Cache[addr].keyItem,
		lock:    set,
	}
	return nil
}

func cleanCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	enable_Cache = true
	defaultAddr_Cache = ""

	for k := range addr_Cache {
		delete(addr_Cache, k)
	}
}

func isValidTopiaAddr(addr tpcrtypes.Address) bool {
	_, err := addr.CryptType()
	if err != nil {
		return false
	}
	return true
}
