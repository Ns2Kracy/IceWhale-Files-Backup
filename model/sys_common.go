package model

type CommonModel struct {
	RuntimePath string
}

type APPModel struct {
	LogPath     string
	LogSaveName string
	LogFileExt  string

	WebDAVPort     string
	WebDAVRootPath string

	BackupRootPath string
}
