package ldf

type BootConfig struct {
	ServerName       string `ldf:"SERVERNAME"`
	PatchServerIP    string `ldf:"PATCHSERVERIP"`
	AuthServerIP     string `ldf:"AUTHSERVERIP"`
	PatchServerPort  int    `ldf:"PATCHSERVERPORT"`
	Logging          int    `ldf:"LOGGING"`
	DataCenterID     uint   `ldf:"DATACENTERID"`
	CPCode           int    `ldf:"CPCODE"`
	AkamaiDLM        bool   `ldf:"AKAMAIDLM"`
	PatchServerDir   string `ldf:"PATCHSERVERDIR"`
	UGCUse3DServices bool   `ldf:"UGCUSE3DSERVICES"`
	UGCServerIP      string `ldf:"UGCSERVERIP"`
	UGCServerDir     string `ldf:"UGCSERVERDIR"`
	PasswordURL      string `ldf:"PASSURL"`
	SigninURL        string `ldf:"SIGNINURL"`
	SignupURL        string `ldf:"SIGNUPURL"`
	RegisterURL      string `ldf:"REGISTERURL"`
	CrashLogURL      string `ldf:"CRASHLOGURL"`
	Locale           string `ldf:"LOCALE"`
	TrackDiskUsage   bool   `ldf:"TRACK_DSK_USAGE"`
}

func DefaultBootConfig() *BootConfig {
	return &BootConfig{
		ServerName:       "Overbuild Universe (US)",
		PatchServerIP:    "localhost",
		AuthServerIP:     "localhost",
		PatchServerPort:  80,
		Logging:          100,
		DataCenterID:     150,
		CPCode:           89164,
		AkamaiDLM:        false,
		PatchServerDir:   "luclient",
		UGCUse3DServices: true,
		UGCServerIP:      "localhost",
		UGCServerDir:     "3dservices",
		PasswordURL:      "https://account.lego.com/en-us/SendPassword.aspx?Username=",
		SigninURL:        "https://account.lego.com/en-us/SignIn.aspx?ReturnUrl=http://universe.lego.com/en-us/myaccount/default.aspx",
		SignupURL:        "http://universe.lego.com/en-us/myaccount/registration/default.aspx",
		RegisterURL:      "https://secure.universe.lego.com/en-us/myaccount/subscription/embeddedlandingpage.aspx?username=",
		CrashLogURL:      "http://services.lego.com/cls.aspx",
		Locale:           "en_US",
		TrackDiskUsage:   true,
	}
}
