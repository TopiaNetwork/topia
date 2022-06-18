package sync

type SyncMode uint32

const (
	FullSync SyncMode = iota
	FastSync
)

type SyncConfig struct {
	Mode SyncMode
}

var DefaultSyncConfig = SyncConfig{
	Mode: FastSync,
}
