package cscore

import (
	"github.com/bellistech/vor/internal/bookmarks"
)

type bookmarkToggleResponse struct {
	Topic      string `json:"topic"`
	Bookmarked bool   `json:"bookmarked"`
}

type bookmarkListResponse struct {
	Bookmarks []string `json:"bookmarks"`
}

// BookmarkToggle adds or removes a bookmark, returning the new state as JSON.
func BookmarkToggle(topic string) string {
	if err := validateTopic(topic); err != nil {
		return errorJSON(err)
	}

	added, err := bookmarks.Toggle(topic)
	if err != nil {
		return errorJSON(err)
	}

	return jsonMarshal(bookmarkToggleResponse{
		Topic:      topic,
		Bookmarked: added,
	})
}

// BookmarkList returns all bookmarks as JSON.
func BookmarkList() string {
	marks := bookmarks.List()
	if marks == nil {
		marks = []string{}
	}
	return jsonMarshal(bookmarkListResponse{Bookmarks: marks})
}

// BookmarkIsStarred returns true if a topic is bookmarked.
func BookmarkIsStarred(topic string) bool {
	return bookmarks.IsBookmarked(topic)
}
