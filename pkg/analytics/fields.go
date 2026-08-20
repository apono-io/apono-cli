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

const eventLogin = "Apono Login"

const (
	OriginInteractive = "interactive mode"
	OriginBrowser     = "browser"
	OriginFlagRun     = "flag run"
)

const (
	eventInteractiveSessionStarted = "Interactive Session Started"
	eventOptionSelected            = "Option Selected"
	eventAccessRequestSubmitted    = "Access Request Submitted"

	surfaceField         = "surface"
	selectIDField        = "select_id"
	optionValueField     = "option_value"
	requestTypeField     = "request_type"
	bundleNameField      = "bundle_name"
	integrationNameField = "integration_name"
	resourceTypeField    = "resource_type"
	resourcesCountField  = "resources_count"
	permissionsField     = "permissions"
	durationField        = "duration"
	justificationField   = "justification"

	maxPropertyValueLength = 1000
	clientTypeCLI          = "CLI"
	maxEventProperties     = 20
)

const (
	InteractiveSurface    = "interactive"
	RequestNewAccessFlow  = "request-new-access"
	ConnectToResourceFlow = "connect-to-resource"
)

const (
	SelectIDActionSelection = "action-selection"
	SelectIDRequestType     = "request-type"
	SelectIDBundle          = "bundle"
	SelectIDIntegration     = "integration"
	SelectIDResourceType    = "resource-type"
	SelectIDResource        = "resource"
	SelectIDPermissions     = "permissions"
	SelectIDAccessSelection = "access-selection"
	SelectIDConnectOption   = "connect-option"
)

const (
	RequestTypeBundle      = "bundle"
	RequestTypeIntegration = "integration"
)

const (
	ConnectOptionConnect        = "connect"
	ConnectOptionInstructions   = "instructions"
	ConnectOptionConnectWithApp = "connect-with-app"
)
