package service

import "github.com/IceWhaleTech/CasaOS-Common/external"

var MyService Services

type Services interface {
	Gateway() external.ManagementService
}

type services struct {
	gateway external.ManagementService
}

func NewService(RuntimePath string) Services {
	gatewayManagement, err := external.NewManagementService(RuntimePath)
	if err != nil && len(RuntimePath) > 0 {
		panic(err)
	}

	return &services{
		gateway: gatewayManagement,
	}
}

func (s *services) Gateway() external.ManagementService {
	return s.gateway
}
