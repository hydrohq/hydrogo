package designsystem

import (
	"net/url"
	"strconv"
)

func adminAuditPageURL(page AdminAuditPageProps, offset int) string {
	values := url.Values{}
	for key, value := range map[string]string{"actor": page.Actor, "action": page.Action, "resource": page.Resource, "target": page.Target, "from": page.From, "until": page.Until} {
		if value != "" {
			values.Set(key, value)
		}
	}
	values.Set("offset", strconv.Itoa(offset))
	return "/admin/audit?" + values.Encode()
}
