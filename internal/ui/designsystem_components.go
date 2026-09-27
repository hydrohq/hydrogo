package ui

import (
	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

type BeforeAfterProps = designsystem.BeforeAfterProps
type HeroProps = designsystem.HeroProps
type CTAProps = designsystem.CTAProps
type PricingItem = designsystem.PricingItem
type PricingTableProps = designsystem.PricingTableProps
type Testimonial = designsystem.Testimonial
type TestimonialsProps = designsystem.TestimonialsProps
type CounterStat = designsystem.CounterStat
type CounterProps = designsystem.CounterProps
type ConfettiProps = designsystem.ConfettiProps
type ContactCTAProps = designsystem.ContactCTAProps
type FAQItem = designsystem.FAQItem
type FAQProps = designsystem.FAQProps
type CopyButtonProps = designsystem.CopyButtonProps
type StickyHeaderProps = designsystem.StickyHeaderProps
type CaptionTrack = designsystem.CaptionTrack
type VideoProps = designsystem.VideoProps
type MasonryItem = designsystem.MasonryItem
type MasonryProps = designsystem.MasonryProps
type Hotspot = designsystem.Hotspot
type HotspotsProps = designsystem.HotspotsProps

func DesignSystemHead() templ.Component                    { return designsystem.Head() }
func BeforeAfter(p BeforeAfterProps) templ.Component       { return designsystem.BeforeAfter(p) }
func HeroBlock(p HeroProps) templ.Component                { return designsystem.HeroBlock(p) }
func CTABlock(p CTAProps) templ.Component                  { return designsystem.CTABlock(p) }
func PricingTable(p PricingTableProps) templ.Component     { return designsystem.PricingTable(p) }
func Testimonials(p TestimonialsProps) templ.Component     { return designsystem.Testimonials(p) }
func AnimatedCounter(p CounterProps) templ.Component       { return designsystem.AnimatedCounter(p) }
func FormSuccessConfetti(p ConfettiProps) templ.Component  { return designsystem.FormSuccessConfetti(p) }
func ContactCTA(p ContactCTAProps) templ.Component         { return designsystem.ContactCTA(p) }
func FAQAccordion(p FAQProps) templ.Component              { return designsystem.FAQAccordion(p) }
func FAQList(p FAQProps) templ.Component                   { return designsystem.FAQList(p) }
func FAQSchema(id string, items []FAQItem) templ.Component { return designsystem.FAQSchema(id, items) }
func CopyButton(p CopyButtonProps) templ.Component         { return designsystem.CopyButton(p) }
func StickyHeader(p StickyHeaderProps) templ.Component     { return designsystem.StickyHeader(p) }
func VideoPlayer(p VideoProps) templ.Component             { return designsystem.VideoPlayer(p) }
func MasonryGrid(p MasonryProps) templ.Component           { return designsystem.MasonryGrid(p) }
func ImageHotspots(p HotspotsProps) templ.Component        { return designsystem.ImageHotspots(p) }
