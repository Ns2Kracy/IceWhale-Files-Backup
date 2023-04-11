package route

import (
	"net/http"

	"github.com/IceWhaleTech/IceWhale-Files-Backup/codegen"
	"github.com/labstack/echo/v4"
)

func (a *api) GetAllBackupFolders(ctx echo.Context) error {
	panic("implement me")
}

func (a *api) GetBackupFoldersByClientID(ctx echo.Context, clientID codegen.ClientIDParam) error {
	panic("implement me")
}

func (a *api) DeleteBackupFolder(ctx echo.Context, clientID codegen.ClientIDParam, params codegen.DeleteBackupFolderParams) error {
	panic("implement me")
}

func (a *api) AddBackupFolder(ctx echo.Context, clientID codegen.ClientIDParam) error {
	var request codegen.BackupFolderRequest
	if err := ctx.Bind(&request); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	// TODO

	return ctx.JSON(http.StatusOK, codegen.ResponseOK{
		// TODO
	})
}
