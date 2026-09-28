package site

import (
	"context"
	"encoding/json"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const BusinessProfileSettingKey = "site.business_profile"

type BusinessHours struct {
	Opens  string `json:"opens,omitempty"`
	Closes string `json:"closes,omitempty"`
	Closed bool   `json:"closed,omitempty"`
}

type BusinessProfile struct {
	Name     string                         `json:"name"`
	Address  string                         `json:"address"`
	Phone    string                         `json:"phone,omitempty"`
	Email    string                         `json:"email,omitempty"`
	Website  string                         `json:"website,omitempty"`
	Timezone string                         `json:"timezone"`
	Hours    map[time.Weekday]BusinessHours `json:"hours,omitempty"`
}

var businessPhone = regexp.MustCompile(`^\+?[0-9(). -]{7,24}$`)

func (p *BusinessProfile) Validate() error {
	if p == nil {
		return ErrInvalidSetting
	}
	p.Name, p.Address = strings.TrimSpace(p.Name), strings.TrimSpace(p.Address)
	p.Phone, p.Email, p.Website = strings.TrimSpace(p.Phone), strings.TrimSpace(p.Email), strings.TrimSpace(p.Website)
	p.Timezone = strings.TrimSpace(p.Timezone)
	if p.Name == "" || len(p.Name) > 200 || p.Address == "" || len(p.Address) > 1000 || (p.Phone != "" && !businessPhone.MatchString(p.Phone)) || (p.Email != "" && !validBusinessEmail(p.Email)) || (p.Website != "" && !validBusinessURL(p.Website)) || len(p.Hours) > 7 {
		return ErrInvalidSetting
	}
	if _, err := time.LoadLocation(p.Timezone); err != nil {
		return ErrInvalidSetting
	}
	for day, hours := range p.Hours {
		if day < time.Sunday || day > time.Saturday {
			return ErrInvalidSetting
		}
		if hours.Closed {
			continue
		}
		if !validClock(hours.Opens) || !validClock(hours.Closes) || hours.Opens == hours.Closes {
			return ErrInvalidSetting
		}
	}
	return nil
}

func (p BusinessProfile) IsOpenAt(at time.Time) bool {
	location, err := time.LoadLocation(p.Timezone)
	if err != nil {
		return false
	}
	local := at.In(location)
	for _, day := range []time.Time{local, local.AddDate(0, 0, -1)} {
		hours, ok := p.Hours[day.Weekday()]
		if !ok || hours.Closed || !validClock(hours.Opens) || !validClock(hours.Closes) {
			continue
		}
		openAt, err := clockOn(day, hours.Opens)
		if err != nil {
			continue
		}
		closeAt, err := clockOn(day, hours.Closes)
		if err != nil {
			continue
		}
		if !closeAt.After(openAt) {
			closeAt = closeAt.Add(24 * time.Hour)
		}
		if !local.Before(openAt) && local.Before(closeAt) {
			return true
		}
	}
	return false
}

func (p BusinessProfile) ToSetting() (Setting, error) {
	if err := p.Validate(); err != nil {
		return Setting{}, err
	}
	data, err := json.Marshal(p)
	if err != nil {
		return Setting{}, err
	}
	return Setting{Key: BusinessProfileSettingKey, Value: string(data), Public: true}, nil
}

func BusinessProfileFromSettings(settings []Setting) (BusinessProfile, bool) {
	for _, setting := range settings {
		if setting.Key == BusinessProfileSettingKey && setting.Public {
			var profile BusinessProfile
			if json.Unmarshal([]byte(setting.Value), &profile) == nil && profile.Validate() == nil {
				return profile, true
			}
		}
	}
	return BusinessProfile{}, false
}

func validClock(value string) bool { _, err := time.Parse("15:04", value); return err == nil }
func clockOn(day time.Time, value string) (time.Time, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(day.Year(), day.Month(), day.Day(), parsed.Hour(), parsed.Minute(), 0, 0, day.Location()), nil
}
func businessHoursLabel(hours BusinessHours) string {
	if hours.Closed {
		return "Kapalı"
	}
	return hours.Opens + "–" + hours.Closes
}
func validBusinessEmail(value string) bool {
	addr, err := mail.ParseAddress(value)
	return err == nil && addr.Address == value && len(value) <= 320
}
func validBusinessURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.IsAbs() && u.Host != "" && (u.Scheme == "https" || u.Scheme == "http") && len(value) <= 2048
}

// SaveBusinessProfile stores the public profile through the existing site
// settings repository, keeping this single source of truth optional.
func (s Service) SaveBusinessProfile(ctx context.Context, profile BusinessProfile) error {
	setting, err := profile.ToSetting()
	if err != nil {
		return err
	}
	return s.SaveSetting(ctx, setting)
}
