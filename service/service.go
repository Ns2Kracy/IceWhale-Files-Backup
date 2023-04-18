package service

import "github.com/IceWhaleTech/CasaOS-Common/external"

var MyService Services

type Services interface {
	Backup() *backup

	Gateway() external.ManagementService
}

type services struct {
	backup  *backup
	gateway external.ManagementService
}

func NewService(RuntimePath string) Services {
	gatewayManagement, err := external.NewManagementService(RuntimePath)
	if err != nil && len(RuntimePath) > 0 {
		panic(err)
	}

	return &services{
		backup:  NewBackupService(),
		gateway: gatewayManagement,
	}
}

func (s *services) Backup() *backup {
	return s.backup
}

func (s *services) Gateway() external.ManagementService {
	return s.gateway
}
