package designsystem

type CalloutProps struct{ Tone, Title, Text string }
type ActionCardProps struct{ Title, Text, ButtonText, ButtonURL string }
type CarouselSlide struct{ Src, Alt, Caption string }
type CarouselProps struct {
	ID, Label string
	Slides    []CarouselSlide
}
type LayoutColumn struct{ Title, Text, ButtonText, ButtonURL string }
type ColumnsProps struct {
	Count   int
	Columns []LayoutColumn
}
type Feature struct{ Icon, Title, Text string }
type FeatureListProps struct {
	Heading   string
	Items     []Feature
	Checklist bool
}
type VideoHeroProps struct{ Title, Subtitle, VideoURL, PosterURL, ButtonText, ButtonURL string }
type TrustBadge struct{ Icon, Title, Subtitle string }
type TrustBarProps struct {
	Heading string
	Items   []TrustBadge
}
type FigureProps struct{ Src, Alt, Caption, Credit, CreditURL string }
type BusinessStatusProps struct {
	Open                          bool
	OpenLabel, ClosedLabel, Hours string
}
type LocalePickerProps struct {
	Locales  []string
	Selected string
}

func calloutTone(value string) string {
	switch value {
	case "warning", "success", "tip":
		return value
	default:
		return "info"
	}
}
func layoutCount(value int) int {
	if value == 3 {
		return 3
	}
	return 2
}
func iconText(value string) string {
	switch value {
	case "shield":
		return "◆"
	case "star":
		return "★"
	case "award":
		return "✦"
	case "check":
		return "✓"
	case "lock":
		return "▣"
	case "heart":
		return "♥"
	case "lightning":
		return "↯"
	default:
		return "•"
	}
}
func businessStatusLabel(p BusinessStatusProps) string {
	if p.Open {
		return label(p.OpenLabel, "Şu anda açığız")
	}
	return label(p.ClosedLabel, "Şu anda kapalıyız")
}
func featureListClass(checklist bool) string {
	if checklist {
		return "ds-feature-list__items ds-feature-list__items--check"
	}
	return "ds-feature-list__items"
}
