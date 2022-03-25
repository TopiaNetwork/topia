package badger

import (
	"testing"

	"github.com/stretchr/testify/assert"

	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

func TestMultiTXs(t *testing.T) {
	log, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.DefaultLogFormat, tplog.DefaultLogOutput, "")

	backend := NewBadgerBackend(log, "test", "./test", 100)
	assert.NotEqual(t, nil, backend)
	//backend.Close()

	verSet, _ := backend.Versions()

	backend.SaveVersion(verSet.Last() + 1)
	verSet, _ = backend.Versions()
	t.Logf("Begin version=%d", verSet.Last())

	t.Log("Tx 1....")
	rw0 := backend.ReadWriter()
	rw0.Set([]byte("test1"), []byte("value1"))
	rw0.Set([]byte("test2"), []byte("value2"))

	t.Log("Tx 2...")
	rw := backend.ReadWriter()
	rw.Set([]byte("test11"), []byte("value11"))
	rw.Set([]byte("test21"), []byte("value21"))

	t.Logf("After tx 2 creating, pending count=%d", backend.PendingTxCount())

	t.Logf("Before tx 2 committing...")
	its0, err := rw.Iterator(nil, nil)
	assert.Equal(t, nil, err)
	for its0.Next() {
		t.Logf("Key=%s, Val0=%s", string(its0.Key()), string(its0.Value()))
	}
	its0.Close()

	err = rw0.Commit()
	assert.Equal(t, nil, err)
	t.Logf("pending count=%d", backend.PendingTxCount())
	err = backend.SaveVersion(verSet.Last() + 1)
	assert.Equal(t, nil, err)

	r0, _ := backend.ReaderAt(verSet.Last() + 1)
	its0, err = r0.Iterator(nil, nil)
	assert.Equal(t, nil, err)
	for its0.Next() {
		t.Logf("Key0=%s, Val0=%s", string(its0.Key()), string(its0.Value()))
	}
	its0.Close()

	verSet, _ = backend.Versions()
	t.Logf("Cur version=%d", verSet.Last())

	err = rw.Commit()
	assert.Equal(t, nil, err)

	err = backend.SaveVersion(verSet.Last() + 1)
	assert.Equal(t, nil, err)
	verSet, _ = backend.Versions()
	t.Logf("Cur version=%d", verSet.Last())

	r, _ := backend.ReaderAt(verSet.Last())
	its, err := r.Iterator(nil, nil)
	assert.Equal(t, nil, err)
	for its.Next() {
		t.Logf("Key=%s, Val=%s", string(its.Key()), string(its.Value()))
	}
	its.Close()
}
