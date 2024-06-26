package selectors

import (
	"time"

	numberinput "github.com/apono-io/apono-cli/pkg/interactive/inputs/number_input"
)

func RunDurationInput(optional bool, minValueInHours float64, maxValueInHours float64) (*time.Duration, error) {
	durationInput := numberinput.NumberInput{
		Title:       "Enter duration in hours",
		PostTitle:   "Duration",
		Placeholder: "Duration",
		Optional:    optional,
		MinValue:    &minValueInHours,
		MaxValue:    &maxValueInHours,
	}

	durationInHours, err := numberinput.LaunchNumberInput(durationInput)
	if err != nil {
		return nil, err
	}

	if durationInHours == nil {
		return nil, nil
	}

	duration := time.Duration(*durationInHours * float64(time.Hour))
	return &duration, nil
}
