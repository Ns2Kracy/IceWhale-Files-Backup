package main

import (
	"net"
	"net/http"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/pkg/config"
	"go.uber.org/zap"
	"golang.org/x/net/webdav"
)

func StartWebDAVService() (*http.Server, chan error) {
	// setup listener
	listener, err := net.Listen("tcp", net.JoinHostPort("", config.AppInfo.WebDAVPort))
	if err != nil {
		panic(err)
	}

	webDAVServerError := make(chan error, 1)
	webDAVServer := &http.Server{
		Handler: &webdav.Handler{
			FileSystem: webdav.Dir(config.AppInfo.DataRootPath),
			LockSystem: webdav.NewMemLS(),
			Logger: func(r *http.Request, err error) {
				if err != nil {
					logger.Error("WebDAV error", zap.Error(err))
					return
				}

				logger.Info("WebDAV request", zap.String("method", r.Method), zap.String("path", r.URL.Path))
			},
		},
		ReadHeaderTimeout: 5 * time.Second, // fix G112: Potential slowloris attack (see https://github.com/securego/gosec)
	}

	go func() {
		webDAVServerError <- webDAVServer.Serve(listener) // not using http.serve() to fix G114: Use of net/http serve function that has no support for setting timeouts (see
	}()

	logger.Info("WebDAV service is listening...", zap.Any("address", listener.Addr().String()))

	return webDAVServer, webDAVServerError
}
