package ui

import (
	"strconv"

	"github.com/ardaozyildirim/hydrogo/internal/forms"
)

type WorkingHoursDay struct {
	ID    string
	Label string
	Open  string
	Close string
}

func WorkingHoursDays(hours forms.WorkingHours) []WorkingHoursDay {
	labels := [...]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	days := make([]WorkingHoursDay, len(hours.Days))
	for index, value := range hours.Days {
		days[index] = WorkingHoursDay{ID: strconv.Itoa(index), Label: labels[index], Open: value.Open, Close: value.Close}
	}
	return days
}
