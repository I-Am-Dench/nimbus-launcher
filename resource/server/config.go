package server

import "github.com/I-Am-Dench/lu-launcher/ldf"

type Config struct {
	SettingDir  string
	DownloadDir string

	Name          string
	PatchToken    string
	PatchProtocol string

	Config *ldf.BootConfig
}
