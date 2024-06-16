package selectors

import (
	numberinput "github.com/apono-io/apono-cli/pkg/interactive/inputs/number_input"
	"time"
)

func RunDurationInput(optional bool, maxValueInHours float64, minValueInHours float64) (*time.Duration, error) {
	durationInput := numberinput.NumberInput{
		Title:       "Enter duration in hours",
		PostTitle:   "Duration",
		Placeholder: "Duration",
		Optional:    optional,
		MaxValue:    &maxValueInHours,
		MinValue:    &minValueInHours,
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
