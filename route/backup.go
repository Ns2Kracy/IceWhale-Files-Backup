package route

import (
	"net/http"

	"github.com/IceWhaleTech/IceWhale-Files-Backup/codegen"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/service"
	"github.com/labstack/echo/v4"
)

func (a *api) GetAllFolderBackups(ctx echo.Context) error {
	panic("implement me")
}

func (a *api) GetFolderBackupsByClientID(ctx echo.Context, clientID codegen.ClientIDParam) error {
	panic("implement me")
}

func (a *api) DeleteFolderBackup(ctx echo.Context, clientID codegen.ClientIDParam, params codegen.DeleteFolderBackupParams) error {
	panic("implement me")
}

func (a *api) RunFolderBackup(ctx echo.Context, clientID codegen.ClientIDParam) error {
	var request codegen.FolderBackupRequest
	if err := ctx.Bind(&request); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	if request.ClientId == nil ||
		request.ClientFolderPath == nil ||
		request.ClientFolderFileSizes == nil ||
		request.ClientFolderFileHashes == nil {
		message := "certain fields are missing in the request body"
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	// TODO - compare with file hashes and only backup the files that have changed/deleted
	folderBackup, err := service.Proceed(request)
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.FolderBackupOK{
		Data: folderBackup,
	})
}
