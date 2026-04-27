// Package mobile provides the gomobile-bindable API for cs.
// All exported functions use only gomobile-safe types:
// string, int, float64, bool, []byte, error.
// No interfaces, channels, func types, or struct pointers.
//
// OFFLINE SACRED LAW: This package imports ZERO network packages.
package mobile

import (
	vor "github.com/bellistech/vor"
	"github.com/bellistech/vor/pkg/cscore"
)

// MobileInit initializes the Go core with embedded sheet data.
// Must be called once from application:didFinishLaunchingWithOptions:
// before any other function in this package.
func MobileInit() error {
	return cscore.Init(vor.EmbeddedSheets, vor.EmbeddedDetails)
}

// MobileSetDataDir sets the app sandbox directory for bookmarks.
func MobileSetDataDir(path string) {
	cscore.SetDataDir(path)
}

// MobileListTopicsJSON returns all topics as a JSON array.
func MobileListTopicsJSON() string {
	return cscore.ListTopicsJSON()
}

// MobileGetSheetJSON returns a single sheet as JSON.
func MobileGetSheetJSON(name string) string {
	return cscore.GetSheetJSON(name)
}

// MobileGetDetailJSON returns a detail page as JSON.
func MobileGetDetailJSON(name string) string {
	return cscore.GetDetailJSON(name)
}

// MobileRandomTopicJSON returns a random sheet as JSON.
func MobileRandomTopicJSON() string {
	return cscore.RandomTopicJSON()
}

// MobileSearchJSON searches sheets and returns results as JSON.
func MobileSearchJSON(query string) string {
	return cscore.SearchJSON(query)
}

// MobileCategoriesJSON returns all categories with counts as JSON.
func MobileCategoriesJSON() string {
	return cscore.CategoriesJSON()
}

// MobileCategoryTopicsJSON returns topics in a category as JSON.
func MobileCategoryTopicsJSON(category string) string {
	return cscore.CategoryTopicsJSON(category)
}

// MobileRelatedJSON returns related topics as JSON.
func MobileRelatedJSON(name string) string {
	return cscore.RelatedJSON(name)
}

// MobileCompareJSON compares two topics as JSON.
func MobileCompareJSON(nameA, nameB string) string {
	return cscore.CompareJSON(nameA, nameB)
}

// MobileLearnPathJSON returns topics ordered by prerequisites as JSON.
func MobileLearnPathJSON(category string) string {
	return cscore.LearnPathJSON(category)
}

// MobileStatsJSON returns statistics as JSON.
func MobileStatsJSON() string {
	return cscore.StatsJSON()
}

// MobileCalcEval evaluates an arithmetic expression and returns JSON.
func MobileCalcEval(expr string) string {
	return cscore.CalcEval(expr)
}

// MobileSubnetCalc parses a CIDR and returns subnet info as JSON.
func MobileSubnetCalc(input string) string {
	return cscore.SubnetCalc(input)
}

// MobileBookmarkToggle toggles a bookmark, returning state as JSON.
func MobileBookmarkToggle(topic string) string {
	return cscore.BookmarkToggle(topic)
}

// MobileBookmarkList returns all bookmarks as JSON.
func MobileBookmarkList() string {
	return cscore.BookmarkList()
}

// MobileBookmarkIsStarred returns true if a topic is bookmarked.
func MobileBookmarkIsStarred(topic string) bool {
	return cscore.BookmarkIsStarred(topic)
}

// MobileVerifyJSON checks math in a detail page and returns JSON.
func MobileVerifyJSON(topic string) string {
	return cscore.VerifyJSON(topic)
}

// MobileRenderMarkdownToHTML converts markdown to HTML as JSON.
func MobileRenderMarkdownToHTML(md string) string {
	return cscore.RenderMarkdownToHTML(md)
}
