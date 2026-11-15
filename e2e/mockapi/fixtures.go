package mockapi

import (
	"time"

	"github.com/apono-io/apono-cli/pkg/clientapi"
)

// Fixture IDs shared by the default state and the tests.
const (
	AccountName = "E2E Account"
	UserName    = "E2E User"
	UserEmail   = "e2e@apono.test"

	PostgresIntegrationID   = "int-postgres"
	PostgresIntegrationType = "postgresql"
	PostgresIntegrationName = "e2e-postgres"
	PostgresResourceTypeID  = "postgresql-database"
	PostgresResourceID      = "res-pg-e2e"
	PostgresResourceSource  = "e2e"
	PostgresPermissionID    = "READ_ONLY"

	KubeIntegrationID   = "int-k8s"
	KubeIntegrationType = "kubernetes"
	KubeIntegrationName = "e2e-kind"

	BundleID   = "bundle-e2e"
	BundleName = "E2E Bundle"

	PostgresSessionID = "session-pg"
	KubeSessionID     = "session-k8s"

	ActiveRequestID = "req-active"
	AccessGroupID   = "group-pg"
)

// Session is a mocked access session plus what the backend returns for its access details.
type Session struct {
	Model     clientapi.AccessSessionClientModel
	RequestID string
	BundleID  string

	// Details is returned by GET access-sessions/{id}/access-details.
	Details clientapi.AccessSessionDetailsClientModel
	// DetailsAfterReset replaces Details once reset-credentials was called, if set.
	DetailsAfterReset *clientapi.AccessSessionDetailsClientModel
	// ResetPolls is how many GET access-sessions/{id} polls after a reset still report the old
	// credentials status, to exercise the CLI's polling loop.
	ResetPolls int

	resetCalled bool
}

// State holds every fixture the mock serves. Handlers mutate it under the server lock.
type State struct {
	UserSession   clientapi.UserSessionClientModel
	Integrations  []clientapi.IntegrationClientModel
	Bundles       []clientapi.BundleClientModel
	ResourceTypes map[string][]clientapi.ResourceTypeClientModel // by integration id
	Resources     []clientapi.ResourceClientModel
	Permissions   map[string][]clientapi.PermissionClientModel // by resource type id
	Sessions      []*Session
	Requests      []*clientapi.AccessRequestClientModel
	AccessUnits   map[string][]clientapi.AccessUnitClientModel // by access group id

	// NewRequestStatus is the status of requests created through POST access-requests.
	NewRequestStatus string
	// DryRun is returned by POST access-requests/dry-run.
	DryRun clientapi.DryRunClientResponse
}

// Session returns the session with the given id, or nil.
func (st *State) Session(id string) *Session {
	for _, s := range st.Sessions {
		if s.Model.Id == id {
			return s
		}
	}
	return nil
}

// Request returns the request with the given id, or nil.
func (st *State) Request(id string) *clientapi.AccessRequestClientModel {
	for _, r := range st.Requests {
		if r.Id == id {
			return r
		}
	}
	return nil
}

// DefaultState returns a small, self-consistent account: a Postgres and a Kubernetes
// integration, one bundle, an active request and one session per integration.
func DefaultState() *State {
	pg := Integration(PostgresIntegrationID, PostgresIntegrationType, PostgresIntegrationName)
	k8s := Integration(KubeIntegrationID, KubeIntegrationType, KubeIntegrationName)
	pgDB := clientapi.ResourceTypeClientModel{
		Id:          PostgresResourceTypeID,
		Name:        "PostgreSQL Database",
		DisplayPath: "Database",
		Icons:       icons(),
	}

	pgResource := clientapi.ResourceClientModel{
		Id:          PostgresResourceID,
		SourceId:    PostgresResourceSource,
		Name:        "e2e",
		Path:        "e2e",
		Type:        pgDB,
		Integration: pg,
	}
	readOnly := clientapi.PermissionClientModel{Id: PostgresPermissionID, Name: "Read only"}

	activeRequest := Request(ActiveRequestID, "Active", pg)

	return &State{
		UserSession: clientapi.UserSessionClientModel{
			User:    clientapi.UserClientModel{Id: "user-e2e", Name: UserName, Email: UserEmail},
			Account: clientapi.AccountClientModel{Id: "account-e2e", Name: AccountName, Plan: "enterprise"},
			Settings: clientapi.UserSettingsClientModel{
				RequestCustomFields: []clientapi.RequestCustomFieldModel{},
			},
		},
		Integrations:  []clientapi.IntegrationClientModel{pg, k8s},
		Bundles:       []clientapi.BundleClientModel{{Id: BundleID, Name: BundleName, Labels: map[string]string{}}},
		ResourceTypes: map[string][]clientapi.ResourceTypeClientModel{PostgresIntegrationID: {pgDB}},
		Resources:     []clientapi.ResourceClientModel{pgResource},
		Permissions:   map[string][]clientapi.PermissionClientModel{PostgresResourceTypeID: {readOnly}},
		Sessions: []*Session{
			NewSession(PostgresSessionID, "e2e postgres", pg, ActiveRequestID,
				`psql "postgresql://apono_e2e:e2e-password@localhost:5432/e2e" -c "select current_user"`),
			NewSession(KubeSessionID, "e2e kind", k8s, ActiveRequestID, "kubectl config use-context e2e"),
		},
		Requests: []*clientapi.AccessRequestClientModel{activeRequest},
		AccessUnits: map[string][]clientapi.AccessUnitClientModel{
			AccessGroupID: {{Id: "unit-pg", Resource: pgResource, Permission: readOnly}},
		},
		NewRequestStatus: "Active",
		DryRun:           clientapi.DryRunClientResponse{Status: "Ok", Errors: []clientapi.DryRunAccessRequestError{}},
	}
}

// Integration builds an integration fixture.
func Integration(id, integrationType, name string) clientapi.IntegrationClientModel {
	return clientapi.IntegrationClientModel{
		Id:              id,
		Type:            integrationType,
		TypeDisplayName: integrationType,
		Name:            name,
		Icons:           icons(),
	}
}

// NewSession builds a session that supports the cli and instructions connection methods and
// whose access details return cliCommand.
func NewSession(id, name string, integration clientapi.IntegrationClientModel, requestID, cliCommand string) *Session {
	credentials := clientapi.AccessSessionClientModelCredentials{Id: "cred-" + id, Status: "shared", CanReset: true}

	return &Session{
		Model: clientapi.AccessSessionClientModel{
			Id:                id,
			Name:              name,
			Type:              clientapi.SessionTypeClientModel{Id: integration.Type, Name: integration.Type, Icons: icons()},
			Integration:       integration,
			Credentials:       *clientapi.NewNullableAccessSessionClientModelCredentials(&credentials),
			ConnectionMethods: []string{"cli", "instructions", "json"},
			Status:            "Active",
			Launchers:         []clientapi.LauncherSummaryClientModel{},
		},
		RequestID: requestID,
		Details:   Details(cliCommand),
	}
}

// Details builds access details whose cli command is cliCommand.
func Details(cliCommand string) clientapi.AccessSessionDetailsClientModel {
	return clientapi.AccessSessionDetailsClientModel{
		Instructions: clientapi.InstructionClientModel{
			Plain:    "Run: " + cliCommand,
			Markdown: "Run: `" + cliCommand + "`",
		},
		Json:      map[string]interface{}{"command": cliCommand},
		Cli:       *clientapi.NewNullableString(&cliCommand),
		Launchers: []clientapi.LauncherClientModel{},
	}
}

// Request builds an access request fixture for one integration.
func Request(id, status string, integration clientapi.IntegrationClientModel) *clientapi.AccessRequestClientModel {
	justification := "e2e test"
	return &clientapi.AccessRequestClientModel{
		Id:            id,
		Requestor:     clientapi.UserClientModel{Id: "user-e2e", Name: UserName, Email: UserEmail},
		CreationTime:  float64(time.Now().Unix()),
		Status:        clientapi.RequestStatusClientModel{Status: status, Metadata: map[string]string{}},
		Justification: *clientapi.NewNullableString(&justification),
		AccessGroups: []clientapi.AccessGroupClientModel{{
			Id:            AccessGroupID,
			Integration:   integration,
			ResourceTypes: []clientapi.ResourceTypeClientModel{},
			Status:        clientapi.AccessGroupStatusClientModel{Status: clientapi.ACCESSGROUPSTATUS_SUCCESS},
		}},
		ExtendOptions: clientapi.ExtendOptionsClientModel{ExtensionAvailability: "unavailable"},
	}
}

func icons() clientapi.IconsConfigClientModel {
	return clientapi.IconsConfigClientModel{Png: "https://example.test/icon.png", Svg: "https://example.test/icon.svg"}
}
