package analytics

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/apono-io/apono-cli/pkg/aponoapi"
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
		ClientType: clientapi.ANALYTICCLIENTTYPECLIENTMODEL_CLI,
		Properties: properties,
	}

	_, _ = client.ClientAPI.AnalyticsAPI.SendAnalyticsEvent(cmd.Context()).CreateAnalyticEventClientModel(req).Execute()
}

func GenerateCommandID() string {
	return uuid.New().String()
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05.000Z07:00")
}
