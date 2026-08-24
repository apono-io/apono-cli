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
	invokedByField       = "invoked_by"
	isInteractiveField   = "is_interactive"
	termProgramField     = "term_program"
	ciField              = "ci"
)

const (
	eventLaunchClientRun = "Command Launch Client Run"

	guiClientField       = "guiClient"
	sessionIDField       = "session_id"
	integrationTypeField = "integrationType"
	originField          = "origin"
)

const eventLogin = "Apono Login"

const (
	OriginInteractive = "interactive mode"
	OriginBrowser     = "browser"
	OriginFlagRun     = "flag run"
)
