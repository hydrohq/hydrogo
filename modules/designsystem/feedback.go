package designsystem

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type HelpfulFeedbackProps struct {
	ContentType string
	ContentID   string
	ElementID   string
	CSRF        string
	Nonce       string
	Up          int64
	Down        int64
	LoadHTMX    bool
}

type FAQFeedbackMetric struct {
	Question string
	Up, Down int64
}

func faqFeedbackProps(p FAQProps, item FAQItem, index int) HelpfulFeedbackProps {
	id := FAQFeedbackID(item, index)
	counts := p.FeedbackCounts[id]
	return HelpfulFeedbackProps{ContentType: "faq", ContentID: id, ElementID: "faq-feedback-" + id, CSRF: p.CSRF, Up: counts[0], Down: counts[1], LoadHTMX: index == 0}
}

func FAQFeedbackID(item FAQItem, index int) string {
	value := fmt.Sprintf("%s\x00%s\x00%d", item.Question, item.Answer, index)
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:16])
}
