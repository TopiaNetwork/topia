package service

type SyncService interface {
	IsSyncing() bool
}

func NewSyncService() SyncService {
	return &syncService{}
}

type syncService struct {
}

func (s *syncService) IsSyncing() bool {
	return true
}
