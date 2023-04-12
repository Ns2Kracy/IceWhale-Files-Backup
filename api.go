package main

import (
	"net"
	"net/http"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/model"
	util_http "github.com/IceWhaleTech/CasaOS-Common/utils/http"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/common"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/route"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/service"
	"go.uber.org/zap"
)

func StartAPIService() (*http.Server, chan error) {
	// setup listener
	listener, err := net.Listen("tcp", net.JoinHostPort(common.Localhost, "0"))
	if err != nil {
		panic(err)
	}

	// initialize routers and register at gateway
	{
		apiPaths := []string{
			route.V2APIPath,
			route.V2DocPath,
		}

		for _, apiPath := range apiPaths {
			if err := service.MyService.Gateway().CreateRoute(&model.Route{
				Path:   apiPath,
				Target: "http://" + listener.Addr().String(),
			}); err != nil {
				panic(err)
			}
		}
	}

	v2Prefix, v2Router := route.InitV2Router()
	v2DocPrefix, v2DocRouter := route.InitV2DocRouter(_docHTML, _docYAML)

	mux := &util_http.HandlerMultiplexer{
		HandlerMap: map[string]http.Handler{
			v2Prefix:    v2Router,
			v2DocPrefix: v2DocRouter,
		},
	}

	apiServerError := make(chan error, 1)
	apiServer := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second, // fix G112: Potential slowloris attack (see https://github.com/securego/gosec)
	}

	go func() {
		apiServerError <- apiServer.Serve(listener) // not using http.serve() to fix G114: Use of net/http serve function that has no support for setting timeouts (see https://github.com/securego/gosec)
	}()

	logger.Info("files backup API service is listening...", zap.Any("address", listener.Addr().String()))

	return apiServer, apiServerError
}
