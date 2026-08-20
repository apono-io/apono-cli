package analytics

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
	"github.com/apono-io/apono-cli/pkg/build"
	"github.com/apono-io/apono-cli/pkg/clientapi"
	"github.com/apono-io/apono-cli/pkg/version"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func SendCommandAnalyticsEvent(cmd *cobra.Command, args []string) {
	endTime := time.Now()
	startTime, err := GetStartTime(cmd.Context())
	if err != nil {
		return
	}

	client, err := aponoapi.GetClient(cmd.Context())
	if err != nil {
		return
	}

	cmdVersion, err := version.GetVersion(cmd.Context())
	if err != nil {
		return
	}

	commandID, err := GetCommandID(cmd.Context())
	if err != nil {
		return
	}

	properties := map[string]interface{}{
		commandIDField:       commandID,
		commandPathField:     cmd.CommandPath(),
		commandArgsField:     args,
		cliVersionField:      cmdVersion.Version,
		operatingSystemField: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		startTimeField:       formatTime(*startTime),
		endTimeField:         formatTime(endTime),
		exitCodeField:        0,
	}

	if shell := os.Getenv("SHELL"); shell != "" {
		properties[shellField] = shell
	}

	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Changed {
			flagKey := flagFieldPrefix + flag.Name
			switch flagValue := flag.Value.(type) {
			case pflag.SliceValue:
				properties[flagKey] = flagValue.GetSlice()
			case pflag.Value:
				properties[flagKey] = flagValue.String()
			default:
				properties[flagKey] = flag.Value.String()
			}
		}
	})

	eventName := fmt.Sprintf("Command %s Ran", cmd.CommandPath())
	req := clientapi.CreateAnalyticEventClientModel{
		EventName:  eventName,
		ClientType: "CLI",
		Properties: properties,
	}

	_, _ = client.ClientAPI.AnalyticsAPI.SendAnalyticsEvent(cmd.Context()).CreateAnalyticEventClientModel(req).Execute()
}

func SendLaunchClientEvent(ctx context.Context, clientID, sessionID, integrationType, origin string) {
	client, err := aponoapi.GetClient(ctx)
	if err != nil {
		return
	}

	req := clientapi.CreateAnalyticEventClientModel{
		EventName:  eventLaunchClientRun,
		ClientType: "CLI",
		Properties: map[string]interface{}{
			guiClientField:       clientID,
			sessionIDField:       sessionID,
			integrationTypeField: integrationType,
			originField:          origin,
		},
	}

	_, _ = client.ClientAPI.AnalyticsAPI.SendAnalyticsEvent(ctx).CreateAnalyticEventClientModel(req).Execute()
}

func SendLoginEvent(ctx context.Context, client *clientapi.APIClient) {
	req := clientapi.CreateAnalyticEventClientModel{
		EventName:  eventLogin,
		ClientType: "CLI",
		Properties: map[string]interface{}{
			cliVersionField:      build.Version,
			operatingSystemField: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		},
	}

	_, _ = client.AnalyticsAPI.SendAnalyticsEvent(ctx).CreateAnalyticEventClientModel(req).Execute()
}

func GenerateCommandID() string {
	return uuid.New().String()
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05.000Z07:00")
}

type AccessRequestSubmittedProperties struct {
	RequestType     string
	BundleName      string
	IntegrationName string
	ResourceType    string
	ResourcesCount  int
	Permissions     []string
	Duration        *time.Duration
	Justification   string
}

func SendInteractiveSessionStartedEvent(ctx context.Context) {
	sendInteractiveEvent(ctx, eventInteractiveSessionStarted, nil)
}

func SendOptionSelectedEvent(ctx context.Context, surface, selectID string, optionValue interface{}) {
	sendInteractiveEvent(ctx, eventOptionSelected, map[string]interface{}{
		surfaceField:     surface,
		selectIDField:    selectID,
		optionValueField: optionValue,
	})
}

func SendAccessRequestSubmittedEvent(ctx context.Context, submitted AccessRequestSubmittedProperties) {
	sendInteractiveEvent(ctx, eventAccessRequestSubmitted, accessRequestSubmittedProperties(submitted))
}

func accessRequestSubmittedProperties(submitted AccessRequestSubmittedProperties) map[string]interface{} {
	permissions := submitted.Permissions
	if permissions == nil {
		permissions = []string{}
	}

	var durationHours float64
	if submitted.Duration != nil {
		durationHours = submitted.Duration.Hours()
	}

	return map[string]interface{}{
		surfaceField:         RequestNewAccessFlow,
		requestTypeField:     submitted.RequestType,
		bundleNameField:      submitted.BundleName,
		integrationNameField: submitted.IntegrationName,
		resourceTypeField:    submitted.ResourceType,
		resourcesCountField:  submitted.ResourcesCount,
		permissionsField:     permissions,
		durationField:        durationHours,
		justificationField:   truncatePropertyValue(submitted.Justification),
	}
}

func sendInteractiveEvent(ctx context.Context, eventName string, properties map[string]interface{}) {
	if !IsInteractiveMode(ctx) {
		return
	}

	client, err := aponoapi.GetClient(ctx)
	if err != nil {
		return
	}

	if properties == nil {
		properties = map[string]interface{}{}
	}
	properties[cliVersionField] = build.Version

	req := clientapi.CreateAnalyticEventClientModel{
		EventName:  eventName,
		ClientType: clientTypeCLI,
		Properties: properties,
	}

	_, _ = client.ClientAPI.AnalyticsAPI.SendAnalyticsEvent(ctx).CreateAnalyticEventClientModel(req).Execute()
}

func truncatePropertyValue(value string) string {
	runes := []rune(value)
	if len(runes) <= maxPropertyValueLength {
		return value
	}

	return string(runes[:maxPropertyValueLength])
}
