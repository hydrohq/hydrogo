package designsystem

type PublicSearchResult struct{ Title, Category, URL string }
type PublicSearchResultsProps struct {
	Term    string
	Results []PublicSearchResult
}
type SearchInsightRow struct {
	Term, Locale                 string
	SearchCount, ZeroResultCount int64
}
type SearchInsightsProps struct {
	RangeDays              int
	Items, ZeroResultItems []SearchInsightRow
}
