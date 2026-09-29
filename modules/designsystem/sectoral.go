package designsystem

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

func tableValue(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}

func formatReadingTime(minutes int) string {
	if minutes == 1 {
		return "1 dk okuma"
	}
	return fmt.Sprintf("%d dk okuma", minutes)
}

type CookieNoticeProps struct {
	Heading, Message, AcceptLabel, RejectLabel, Action, CSRF, Variant, ReturnTo string
}

type EmailDNSCheck struct {
	Name, Detail, Guidance string
	OK                     bool
}

type TokenLifetimeFormProps struct {
	CSRF, Reset, Invitation string
	Saved                   bool
}

type EmailBrandingFormProps struct {
	CSRF, LegalName, PhysicalAddress, PrivacyPolicyURL  string
	FooterNotice, FooterText, LogoAssetID, PrimaryColor string
	Saved                                               bool
}

type DataTableProps struct {
	Caption string
	Columns []string
	Rows    [][]string
}

type SocialPhoto struct{ Image, Alt, URL string }
type SocialPhotoGridProps struct {
	Label  string
	Photos []SocialPhoto
}

var markupTags = regexp.MustCompile(`<[^>]*>`)

func ReadingTime(value string) int {
	words := len(strings.Fields(html.UnescapeString(markupTags.ReplaceAllString(value, " "))))
	minutes := (words + 199) / 200
	if minutes < 1 {
		return 1
	}
	return minutes
}

func SmartTypography(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "...", "…")
	value = strings.ReplaceAll(value, "--", "—")
	value = strings.ReplaceAll(value, "  ", " ")
	return value
}

func cookieVariant(value string) string {
	switch value {
	case "card", "pill":
		return value
	default:
		return "bar"
	}
}

func safeConsentAction(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.ContainsAny(value, "\\\r\n\x00") {
		return ""
	}
	return value
}
