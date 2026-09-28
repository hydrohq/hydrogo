package forms

import (
	"errors"
	"strings"
	"time"
)

type DailyHours struct {
	Open  string `json:"open,omitempty"`
	Close string `json:"close,omitempty"`
}
type WorkingHours struct {
	Timezone string        `json:"timezone"`
	Days     [7]DailyHours `json:"days"`
	Notice   string        `json:"notice,omitempty"`
}

func (h WorkingHours) Validate() error {
	if len(h.Timezone) > 64 {
		return errors.New("invalid working-hours timezone")
	}
	if _, err := time.LoadLocation(h.Timezone); err != nil {
		return errors.New("invalid working-hours timezone")
	}
	for _, day := range h.Days {
		if day.Open == "" && day.Close == "" {
			continue
		}
		open, e1 := time.Parse("15:04", day.Open)
		close, e2 := time.Parse("15:04", day.Close)
		if e1 != nil || e2 != nil || !close.After(open) {
			return errors.New("working hours must use an increasing HH:MM range")
		}
	}
	if len(h.Notice) > 500 {
		return errors.New("working-hours notice is too long")
	}
	return nil
}

// NextOpening returns zero while the configured local schedule is open.
func (h WorkingHours) NextOpening(now time.Time) (time.Time, error) {
	if err := h.Validate(); err != nil {
		return time.Time{}, err
	}
	loc, _ := time.LoadLocation(h.Timezone)
	local := now.In(loc)
	today := int(local.Weekday())
	if hours := h.Days[today]; hours.Open != "" {
		open, _ := time.Parse("15:04", hours.Open)
		close, _ := time.Parse("15:04", hours.Close)
		opening := time.Date(local.Year(), local.Month(), local.Day(), open.Hour(), open.Minute(), 0, 0, loc)
		closing := time.Date(local.Year(), local.Month(), local.Day(), close.Hour(), close.Minute(), 0, 0, loc)
		if !local.Before(opening) && local.Before(closing) {
			return time.Time{}, nil
		}
		if local.Before(opening) {
			return opening, nil
		}
	}
	for offset := 1; offset <= 7; offset++ {
		date := local.AddDate(0, 0, offset)
		hours := h.Days[int(date.Weekday())]
		if hours.Open == "" {
			continue
		}
		open, _ := time.Parse("15:04", hours.Open)
		return time.Date(date.Year(), date.Month(), date.Day(), open.Hour(), open.Minute(), 0, 0, loc), nil
	}
	return time.Time{}, errors.New("no future working hours configured")
}

func (h WorkingHours) AfterHoursNotice(now time.Time) (string, bool, error) {
	next, err := h.NextOpening(now)
	if err != nil {
		return "", false, err
	}
	if next.IsZero() {
		return "", false, nil
	}
	loc, _ := time.LoadLocation(h.Timezone)
	message := strings.TrimSpace(h.Notice)
	if message == "" {
		message = "We are outside office hours. We will respond when we reopen on {weekday} at {time}."
	}
	message = strings.ReplaceAll(message, "{weekday}", next.In(loc).Format("Monday"))
	message = strings.ReplaceAll(message, "{time}", next.In(loc).Format("15:04"))
	return message, true, nil
}
