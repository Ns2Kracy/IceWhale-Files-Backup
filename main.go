//go:generate bash -c "mkdir -p codegen && go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.12.4 -generate types,server,spec -package codegen api/openapi.yaml > codegen/api.go"

package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/common"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/pkg/config"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/service"
	"github.com/coreos/go-systemd/daemon"
	"go.uber.org/zap"
)

var (
	commit = "private build"
	date   = "private build"

	//go:embed api/index.html
	_docHTML string

	//go:embed api/openapi.yaml
	_docYAML string
)

func main() {
	// parse arguments and intialize
	{
		configFlag := flag.String("c", "", "config file path")
		versionFlag := flag.Bool("v", false, "version")

		flag.Parse()

		if *versionFlag {
			fmt.Printf("v%s\n", common.FilesBackupVersion)
			os.Exit(0)
		}

		println("git commit:", commit)
		println("build date:", date)

		config.InitSetup(*configFlag)

		logger.LogInit(config.AppInfo.LogPath, config.AppInfo.LogSaveName, config.AppInfo.LogFileExt)

		service.MyService = service.NewService(config.CommonInfo.RuntimePath)
	}

	apiService, apiServiceError := StartAPIService()
	webdavService, webdavServiceError := StartWebDAVService()

	// notify systemd that we are ready
	{
		if supported, err := daemon.SdNotify(false, daemon.SdNotifyReady); err != nil {
			logger.Error("Failed to notify systemd that files backup service is ready", zap.Any("error", err))
		} else if supported {
			logger.Info("Notified systemd that files backup service is ready")
		} else {
			logger.Info("This process is not running as a systemd service.")
		}
	}

	// Set up a channel to catch the Ctrl+C signal (SIGINT)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Wait for the signal or server error
	select {
	case <-signalChan:
		fmt.Println("\nReceived signal, shutting down server...")
	case err := <-apiServiceError:
		fmt.Printf("Error starting API service: %s\n", err)
		if err != http.ErrServerClosed {
			os.Exit(1)
		}
	case err := <-webdavServiceError:
		fmt.Printf("Error starting WebDAV service: %s\n", err)
		if err != http.ErrServerClosed {
			os.Exit(1)
		}
	}

	// Create a context with a timeout to allow the server to shut down gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown the apiService
	if err := apiService.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown api server", zap.Any("error", err))
		os.Exit(1)
	}

	// Shutdown the webdavService
	if err := webdavService.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown webdav server", zap.Any("error", err))
		os.Exit(1)
	}
}
