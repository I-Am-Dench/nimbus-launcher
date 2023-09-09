package luconfig

type LUConfig struct {
	ServerName       string `lucfg:"SERVERNAME"`
	PatchServerIP    string `lucfg:"PATCHSERVERIP"`
	AuthServerIP     string `lucfg:"AUTHSERVERIP"`
	PatchServerPort  int32  `lucfg:"PATCHSERVERPORT"`
	Logging          int64  `lucfg:"LOGGING"`
	DataCenterID     int64  `lucfg:"DATACENTERID"`
	CPCode           int32  `lucfg:"CPCODE"`
	AkamaiDLM        bool   `lucfg:"AKAMAIDLM"`
	PatchServerDir   string `lucfg:"PATCHSERVERDIR"`
	UGCUse3DServices bool   `lucfg:"UGCUSE3DSERVICES"`
	UGCServerIP      string `lucfg:"UGCSERVERIP"`
	UGCServerDir     string `lucfg:"UGCSERVERDIR"`
	PasswordURL      string `lucfg:"PASSURL"`
	SigninURL        string `lucfg:"SIGNINURL"`
	SignupURL        string `lucfg:"SIGNUPURL"`
	RegisterURL      string `lucfg:"REGISTERURL"`
	CrashLogURL      string `lucfg:"CRASHLOGURL"`
	Locale           string `lucfg:"LOCALE"`
	TrackDiskUsage   bool   `lucfg:"TRACK_DSK_USAGE"`
}
