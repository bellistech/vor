package main

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/bellistech/vor/internal/registry"
)

// /api/search returns at most limit lines, and with per_topic set at most
// that many lines from any one topic. Without per_topic one long topic can
// fill the whole answer: a sheet with a matching line in 60 sections takes
// all 50 slots and the next topic never shows. With category set, only
// topics in exactly that category are searched, so a small category is not
// crowded out by a large one.
const (
	defaultSearchLimit = 50
	maxSearchLimit     = 500
	maxPerTopic        = 50
)

// searchLimits reads limit and per_topic from a /api/search query. Absent:
// limit 50, per_topic 0 (no cap). Anything else must be a whole number in
// range, or the request is refused.
func searchLimits(q url.Values) (limit, perTopic int, err error) {
	limit, err = intParam(q, "limit", defaultSearchLimit, 1, maxSearchLimit)
	if err != nil {
		return 0, 0, err
	}
	perTopic, err = intParam(q, "per_topic", 0, 1, maxPerTopic)
	if err != nil {
		return 0, 0, err
	}
	return limit, perTopic, nil
}

func intParam(q url.Values, name string, def, lo, hi int) (int, error) {
	if !q.Has(name) {
		return def, nil
	}
	n, err := strconv.Atoi(q.Get(name))
	if err != nil || n < lo || n > hi {
		return 0, fmt.Errorf("%s must be a whole number from %d to %d", name, lo, hi)
	}
	return n, nil
}

// limitMatches keeps the first line of each topic section, at most perTopic
// lines per topic when perTopic > 0, and at most limit lines in all, in the
// order Search ranked them. With category set it keeps only topics in
// exactly that category. A topic is the sheet itself, not its name: names
// repeat across categories.
func limitMatches(matches []registry.Match, category string, limit, perTopic int) []registry.Match {
	type key struct {
		sheet   *registry.Sheet
		section string
	}
	seen := make(map[key]bool)
	perSheet := make(map[*registry.Sheet]int)
	var out []registry.Match
	for _, m := range matches {
		if len(out) >= limit {
			break
		}
		if category != "" && m.Sheet.Category != category {
			continue
		}
		k := key{m.Sheet, m.Section}
		if seen[k] || (perTopic > 0 && perSheet[m.Sheet] >= perTopic) {
			continue
		}
		seen[k] = true
		perSheet[m.Sheet]++
		out = append(out, m)
	}
	return out
}
