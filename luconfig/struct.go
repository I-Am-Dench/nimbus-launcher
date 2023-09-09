package luconfig

type LUConfig struct {
	ServerName       string `lucfg:"SERVERNAME"`
	PatchServerIP    string `lucfg:"PATCHSERVERIP"`
	AuthServerIP     string `lucfg:"AUTHSERVERIP"`
	PatchServerPort  int32  `lucfg:"PATCHSERVERPORT"`
	Logging          int32  `lucfg:"LOGGING"`
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

func New() *LUConfig {
	config := new(LUConfig)
	return config
}

func DefaultConfig() *LUConfig {
	config := New()

	*config = LUConfig{
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

	return config
}
