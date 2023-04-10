package config

import (
	"log"

	"github.com/IceWhaleTech/IceWhale-Files-Backup/common"
	"github.com/IceWhaleTech/IceWhale-Files-Backup/model"
	"gopkg.in/ini.v1"
)

var (
	CommonInfo = &model.CommonModel{
		RuntimePath: "/var/run/icewhale",
	}

	AppInfo = &model.APPModel{
		LogPath:     "/var/log/icewhale",
		LogSaveName: common.FilesBackupServiceName,
		LogFileExt:  "log",
	}

	Cfg            *ini.File
	ConfigFilePath string
)

func InitSetup(config string) {
	ConfigFilePath = FilesBackupConfigFilePath
	if len(config) > 0 {
		ConfigFilePath = config
	}

	var err error

	Cfg, err = ini.Load(ConfigFilePath)
	if err != nil {
		panic(err)
	}

	mapTo("common", CommonInfo)
}

func mapTo(section string, v interface{}) {
	err := Cfg.Section(section).MapTo(v)
	if err != nil {
		log.Fatalf("Cfg.MapTo %s err: %v", section, err)
	}
}
