package designsystem

func StaffBroadcastNoticeClass(severity string) string {
	switch severity {
	case "critical":
		return "border-b border-red-500 bg-red-100 px-4 py-3 text-center font-semibold text-red-950"
	case "warning":
		return "border-b border-amber-500 bg-amber-100 px-4 py-3 text-center font-semibold text-amber-950"
	default:
		return "border-b border-sky-500 bg-sky-100 px-4 py-3 text-center font-semibold text-sky-950"
	}
}

func StaffBroadcastNoticeRole(severity string) string {
	if severity == "critical" {
		return "alert"
	}
	return "status"
}
