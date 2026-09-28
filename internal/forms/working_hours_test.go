package forms

import (
	"testing"
	"time"
)

func TestWorkingHoursDetectsAfterHoursAcrossTimezoneAndWeekend(t *testing.T) {
	var days [7]DailyHours
	days[1] = DailyHours{Open: "09:00", Close: "17:00"}
	days[2] = days[1]
	days[3] = days[1]
	days[4] = days[1]
	days[5] = days[1]
	h := WorkingHours{Timezone: "Europe/Istanbul", Days: days, Notice: "Respond on {weekday} at {time}."}
	now := time.Date(2026, time.September, 26, 12, 0, 0, 0, time.UTC)
	message, after, err := h.AfterHoursNotice(now)
	if err != nil || !after || message != "Respond on Monday at 09:00." {
		t.Fatalf("notice=%q after=%v err=%v", message, after, err)
	}
	now = time.Date(2026, time.September, 22, 10, 0, 0, 0, time.FixedZone("UTC+3", 3*3600))
	_, after, err = h.AfterHoursNotice(now)
	if err != nil || after {
		t.Fatalf("within-hours after=%v err=%v", after, err)
	}
}

func TestWorkingHoursRejectsInvalidIntervals(t *testing.T) {
	var days [7]DailyHours
	days[1] = DailyHours{Open: "17:00", Close: "09:00"}
	if (WorkingHours{Timezone: "UTC", Days: days}).Validate() == nil {
		t.Fatal("invalid time interval accepted")
	}
}

func TestFormAutoresponderAddsNoticeOnlyAfterHours(t *testing.T) {
	var days [7]DailyHours
	for weekday := time.Monday; weekday <= time.Friday; weekday++ {
		days[weekday] = DailyHours{Open: "09:00", Close: "17:00"}
	}
	f := Form{AutoresponderBody: "Thank you.", WorkingHours: &WorkingHours{Timezone: "UTC", Days: days, Notice: "We reopen {weekday} at {time}."}}
	inside, err := f.AutoresponderBodyFor(time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC))
	if err != nil || inside != "Thank you." {
		t.Fatalf("inside=%q err=%v", inside, err)
	}
	outside, err := f.AutoresponderBodyFor(time.Date(2026, 9, 28, 18, 0, 0, 0, time.UTC))
	if err != nil || outside != "Thank you.\n\nWe reopen Tuesday at 09:00." {
		t.Fatalf("outside=%q err=%v", outside, err)
	}
}

func TestPublicWorkingHoursNoticeIsContextAware(t *testing.T) {
	var days [7]DailyHours
	days[time.Monday] = DailyHours{Open: "09:00", Close: "17:00"}
	form := Form{WorkingHours: &WorkingHours{Timezone: "UTC", Days: days, Notice: "We reopen {weekday} at {time}."}}
	inside := form.PublicWorkingHoursNotice(time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC))
	outside := form.PublicWorkingHoursNotice(time.Date(2026, 9, 28, 18, 0, 0, 0, time.UTC))
	if inside != "" || outside != "We reopen Monday at 09:00." {
		t.Fatalf("inside=%q outside=%q", inside, outside)
	}
}
