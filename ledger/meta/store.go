package meta

import (
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
	tplog "github.com/TopiaNetwork/topia/log"
)

type metaStoreSpecification struct {
	log tplog.Logger
	tplgcmm.DBReadWriter
}

func newMetaStoreSpecification(log tplog.Logger, backendRW tplgcmm.DBReadWriter, name string, cacheSize int) *metaStoreSpecification {
	backendDBNamed := backend.NewBackendRWPrefixed([]byte(name), backendRW)

	return &metaStoreSpecification{
		log:          log,
		DBReadWriter: backendDBNamed,
	}
}

func (store *metaStoreSpecification) IterateAllStateDataCB(iterCBFunc IterMetaDataCBFunc) error {
	dataIt, err := store.Iterator(nil, nil)
	if err != nil {
		return err
	}
	defer dataIt.Close()

	for dataIt.Next() {
		iterCBFunc(dataIt.Key(), dataIt.Value())
	}

	return nil
}
