package agent

type Config struct {
	SubmitServerHost 		string
	SubmitServerPort 		int
	SubmitServerUser		string
	SubmitServerPassword	string
	SubmitFsHost			string
	SubmitFsPort			int
	SubmitFsUser			string
	SubmitFsPassword		string
	CacheDir				string
	ConfFile				string
}
