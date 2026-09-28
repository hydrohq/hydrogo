package announcements

import "time"

func (v Announcement) VisibleAt(now time.Time) bool {
	if v.Status != "published" {
		return false
	}
	return (v.StartsAt.IsZero() || !now.Before(v.StartsAt)) && (v.EndsAt.IsZero() || now.Before(v.EndsAt))
}
