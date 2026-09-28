// Package designsystem contains dependency-light public design components.
package designsystem

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type BeforeAfterProps struct {
	ID, BeforeSrc, AfterSrc, BeforeAlt, AfterAlt string
	BeforeLabel, AfterLabel                      string
	Position                                     int
	PositionSet                                  bool
}
type HeroProps struct{ Title, Subtitle, MediaURL, MediaAlt, MediaAssetID, ButtonText, ButtonURL string }
type CTAProps struct{ Title, ButtonText, ButtonURL string }
type PricingItem struct {
	Name       string   `json:"name"`
	PriceCents int64    `json:"price_cents"`
	Currency   string   `json:"currency"`
	Features   []string `json:"features"`
}
type PricingTableProps struct{ Items []PricingItem }
type Testimonial struct {
	Name  string `json:"name"`
	Quote string `json:"quote"`
}
type TestimonialsProps struct{ Items []Testimonial }

type CounterStat struct {
	Value  float64 `json:"number_value"`
	Prefix string  `json:"prefix"`
	Suffix string  `json:"suffix"`
	Label  string  `json:"label"`
	Badge  string  `json:"icon_badge"`
}

type CounterProps struct {
	ID    string
	Stats []CounterStat
}
type ConfettiProps struct {
	Enabled bool
	Label   string
}
type ContactCTAProps struct{ Phone, WhatsApp, Message, PhoneLabel, WhatsAppLabel string }
type CountdownProps struct {
	Target          string `json:"target_datetime"`
	Timezone        string `json:"timezone"`
	ExpiredHeadline string `json:"expired_headline"`
	ExpiredURL      string `json:"expired_cta_url"`
}
type FAQItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}
type FAQProps struct {
	ID, Heading string
	Items       []FAQItem
}
type CopyButtonProps struct {
	Text, Label, CopiedLabel string
	IconOnly                 bool
}
type StickyHeaderProps struct {
	ID        string
	Threshold int
}
type CaptionTrack struct {
	Src, Language, Label string
	Default              bool
}
type VideoProps struct {
	ID, Src, Type, Poster, Label string
	Tracks                       []CaptionTrack
}
type MasonryItem struct{ Image, Alt, Caption, Link string }
type MasonryProps struct {
	ID, Label string
	Columns   int
	Items     []MasonryItem
}
type Hotspot struct {
	X, Y                                float64
	Title, Description, Link, LinkLabel string
}
type HotspotsProps struct {
	ID, Image, Alt, Label string
	Points                []Hotspot
}
type GalleryImage struct{ Src, Alt, Caption string }
type LightboxProps struct {
	ID, Label string
	Images    []GalleryImage
}
type AudioProps struct{ ID, Src, Title, TranscriptURL string }
type ShareProps struct{ URL, Title string }
type InitialsAvatarProps struct{ Initials, Class string }

func Initials(identity string) string {
	var words []string
	var current []rune
	for _, r := range strings.TrimSpace(identity) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current = append(current, r)
		} else if len(current) > 0 {
			words = append(words, string(current))
			current = nil
		}
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	if len(words) == 0 {
		return "?"
	}
	first := []rune(words[0])
	result := strings.ToUpper(string(first[0]))
	if len(words) > 1 {
		second := []rune(words[1])
		result += strings.ToUpper(string(second[0]))
	} else if len(first) > 1 {
		result += strings.ToUpper(string(first[1]))
	}
	return result
}

func InitialsAvatarClass(identity string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(strings.ToLower(strings.TrimSpace(identity))))
	return fmt.Sprintf("ds-avatar--%d", hash.Sum32()%8)
}

var phoneDigits = regexp.MustCompile(`[^0-9+]`)

func NormalizePhone(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "+") {
		return "+" + strings.ReplaceAll(phoneDigits.ReplaceAllString(value[1:], ""), "+", "")
	}
	return phoneDigits.ReplaceAllString(value, "")
}

func safePhoneInput(value string) bool {
	value = strings.TrimSpace(value)
	for i, r := range value {
		if r == '+' && i == 0 {
			continue
		}
		if r >= '0' && r <= '9' || r == ' ' || r == '(' || r == ')' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func WhatsAppURL(number, message string) string {
	if !safePhoneInput(number) {
		return ""
	}
	number = strings.TrimPrefix(NormalizePhone(number), "+")
	if len(number) < 7 || len(number) > 15 || strings.HasPrefix(number, "0") {
		return ""
	}
	for _, r := range number {
		if !unicode.IsDigit(r) {
			return ""
		}
	}
	u := "https://wa.me/" + number
	if message != "" {
		u += "?text=" + url.QueryEscape(message)
	}
	return u
}

func validMediaURL(value string) string {
	value = strings.TrimSpace(value)
	u, err := url.Parse(value)
	if err != nil || value == "" || strings.HasPrefix(value, "//") || u.Scheme != "" && ((u.Scheme != "https" && u.Scheme != "http") || u.Host == "") || u.Scheme == "" && u.Host != "" {
		return ""
	}
	return value
}

func validShareURL(value string) string {
	value = strings.TrimSpace(value)
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" && u.Scheme != "http" || u.Host == "" || u.User != nil {
		return ""
	}
	return u.String()
}

func shareIntentURL(provider, value, title string) string {
	value = validShareURL(value)
	if value == "" {
		return ""
	}
	switch provider {
	case "linkedin":
		return "https://www.linkedin.com/sharing/share-offsite/?url=" + url.QueryEscape(value)
	case "whatsapp":
		return "https://wa.me/?text=" + url.QueryEscape(strings.TrimSpace(title)+" "+value)
	case "email":
		return "mailto:?subject=" + url.QueryEscape(title) + "&body=" + url.QueryEscape(value)
	default:
		return ""
	}
}

func validPosition(value int) int { return max(0, min(100, value)) }
func validFloatPosition(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Max(0, math.Min(100, value))
}
func hotspotStyle(point Hotspot) string {
	return fmt.Sprintf("left:%.2f%%;top:%.2f%%", validFloatPosition(point.X), validFloatPosition(point.Y))
}

func fmtPosition(value int) string { return fmt.Sprint(validPosition(value)) }
func fmtInteger(value int) string  { return strconv.Itoa(max(0, min(10000, value))) }
func fmtFloat(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return "0"
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}
func beforeAfterPosition(p BeforeAfterProps) string {
	if !p.PositionSet && p.Position == 0 {
		return "50"
	}
	return fmtPosition(p.Position)
}
func formatPrice(cents int64) string { return strconv.FormatInt(cents, 10) }
func label(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
func safeID(value string) string {
	var out strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			out.WriteRune(r)
		}
	}
	if out.Len() == 0 {
		return "faq"
	}
	return out.String()
}
func validPhone(value string) string {
	if !safePhoneInput(value) {
		return ""
	}
	normalized := NormalizePhone(value)
	digits := strings.TrimPrefix(normalized, "+")
	if len(digits) < 7 || len(digits) > 15 {
		return ""
	}
	for _, r := range digits {
		if !unicode.IsDigit(r) {
			return ""
		}
	}
	return normalized
}
func videoType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "video/mp4", "video/webm":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "video/mp4"
	}
}
func clampColumns(value int) int { return max(1, min(6, value)) }
func safeLink(value string) string {
	value = strings.TrimSpace(value)
	u, err := url.Parse(value)
	if err != nil || value == "" || strings.HasPrefix(value, "//") {
		return ""
	}
	if u.Scheme == "" || u.Scheme == "https" || u.Scheme == "http" {
		return value
	}
	return ""
}

func faqJSONLD(items []FAQItem) json.RawMessage {
	type answer struct {
		Type string `json:"@type"`
		Text string `json:"text"`
	}
	type question struct {
		Type     string `json:"@type"`
		Name     string `json:"name"`
		Accepted answer `json:"acceptedAnswer"`
	}
	questions := make([]question, 0, len(items))
	for _, item := range items {
		questions = append(questions, question{Type: "Question", Name: item.Question, Accepted: answer{Type: "Answer", Text: item.Answer}})
	}
	data, _ := json.Marshal(struct {
		Context string     `json:"@context"`
		Type    string     `json:"@type"`
		Main    []question `json:"mainEntity"`
	}{"https://schema.org", "FAQPage", questions})
	return json.RawMessage(data)
}

func hasCaption(tracks []CaptionTrack) bool {
	for _, track := range tracks {
		if strings.TrimSpace(track.Src) != "" {
			return true
		}
	}
	return false
}
