package analytics

const (
	commandIDField       = "command_id"
	commandPathField     = "command_path"
	commandArgsField     = "command_args"
	cliVersionField      = "cli_version"
	operatingSystemField = "operating_system"
	shellField           = "shell"
	startTimeField       = "start_time"
	endTimeField         = "end_time"
	exitCodeField        = "exit_code"
	flagFieldPrefix      = "flag_"
)

const (
	eventLaunchClientRun = "Command Launch Client Run"

	guiClientField       = "guiClient"
	sessionIDField       = "session_id"
	integrationTypeField = "integrationType"
	originField          = "origin"
)

const (
	OriginInteractive = "interactive mode"
	OriginBrowser     = "browser"
	OriginFlagRun     = "flag run"
)
