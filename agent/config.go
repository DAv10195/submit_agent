package agent

// agent configuration
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
	MaxRunningTasks			int
	MossParserHost			string
	MossParserPort			int
}
