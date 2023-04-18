package route

import (
	"context"
	"fmt"
	"net/http"

	"github.com/IceWhaleTech/IceWhale-Files-Backup/codegen"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/service"
	"github.com/labstack/echo/v4"
)

func (a *api) GetAllFolderBackups(ctx echo.Context, params codegen.GetAllFolderBackupsParams) error {
	full := params.Full != nil && *params.Full

	allBackups, err := service.MyService.Backup().GetAllBackups(ctx.Request().Context(), full)
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.AllFolderBackupsOK{
		Data: &allBackups,
	})
}

func (a *api) GetFolderBackupsByClientID(ctx echo.Context, clientID codegen.ClientIDParam, params codegen.GetFolderBackupsByClientIDParams) error {
	if clientID == "" {
		message := "client id is missing"
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	isExists, err := service.MyService.Backup().IsClientIDExists(string(clientID))
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	if !isExists {
		message := fmt.Sprintf("no backup found for this client id %s", clientID)
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
	}

	full := params.Full != nil && *params.Full

	backups, err := service.MyService.Backup().GetBackupsByClientID(ctx.Request().Context(), string(clientID), full)
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.FolderBackupsOK{
		Data: &backups,
	})
}

func (a *api) DeleteFolderBackup(ctx echo.Context, clientID codegen.ClientIDParam, params codegen.DeleteFolderBackupParams) error {
	if clientID == "" {
		message := "client id is missing"
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	if params.ClientFolderPath == "" {
		message := "client folder path is missing"
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	backupExists, err := service.MyService.Backup().IsBackupExists(string(clientID), params.ClientFolderPath)
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	if !backupExists {
		message := fmt.Sprintf("no backup found for this client id %s and client folder path %s", clientID, params.ClientFolderPath)
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
	}

	if err := service.MyService.Backup().DeleteBackupsByClientID(context.Background(), string(clientID), params.ClientFolderPath); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	message := fmt.Sprintf("backup for client id %s and client folder path %s has been deleted", clientID, params.ClientFolderPath)

	return ctx.JSON(http.StatusOK, codegen.ResponseOK{
		Message: &message,
	})
}

func (a *api) RunFolderBackup(ctx echo.Context, clientID codegen.ClientIDParam) error {
	var request codegen.FolderBackupRequest
	if err := ctx.Bind(&request); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	if clientID == "" ||
		request.ClientFolderPath == nil ||
		request.ClientFolderFileSizes == nil ||
		request.ClientFolderFileHashes == nil {
		message := "certain fields are missing in the request body"
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	request.ClientID = &clientID

	// compare with file sizes/hashes and only backup the files that have changed/deleted
	folderBackup, err := service.MyService.Backup().Proceed(request)
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.FolderBackupOK{
		Data: folderBackup,
	})
}
