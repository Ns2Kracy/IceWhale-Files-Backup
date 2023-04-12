package route

import (
	"net/http"

	"github.com/IceWhaleTech/IceWhale-Files-Backup/codegen"
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

func (a *api) AddFolderBackup(ctx echo.Context, clientID codegen.ClientIDParam) error {
	var request codegen.FolderBackupRequest
	if err := ctx.Bind(&request); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	// TODO

	return ctx.JSON(http.StatusOK, codegen.ResponseOK{
		// TODO
	})
}
