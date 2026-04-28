package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	vor "github.com/bellistech/vor"
	"github.com/bellistech/vor/internal/bookmarks"
	"github.com/bellistech/vor/internal/calc"
	"github.com/bellistech/vor/internal/custom"
	"github.com/bellistech/vor/internal/registry"
	"github.com/bellistech/vor/internal/render"
	"github.com/bellistech/vor/internal/secrets"
	"github.com/bellistech/vor/internal/stackoverflow"
	"github.com/bellistech/vor/internal/subnet"
	"github.com/bellistech/vor/internal/tui"
	"github.com/bellistech/vor/internal/verify"
)

var version = "dev"

// progName returns the basename used to invoke the binary. The same binary
// is shipped as both `vor` (canonical) and `cs` (legacy alias via symlink);
// help text and error messages match whichever name the user typed.
func progName() string {
	if len(os.Args) == 0 {
		return "vor"
	}
	name := filepath.Base(os.Args[0])
	// strip Windows .exe if present, just in case
	name = strings.TrimSuffix(name, ".exe")
	if name == "" {
		return "vor"
	}
	return name
}

func main() {
	prog := progName()
	search := flag.String("s", "", "search across all cheatsheets")
	detail := flag.String("d", "", "show deep theory/math for topic")
	list := flag.Bool("l", false, "list all topics with descriptions")
	add := flag.String("add", "", "add custom cheatsheet from file")
	force := flag.Bool("force", false, "with --add, overwrite an existing custom sheet")
	edit := flag.String("edit", "", "open topic in $EDITOR for customization")
	ver := flag.Bool("v", false, "print version")
	random := flag.Bool("random", false, "show a random cheatsheet")
	count := flag.Bool("count", false, "show sheet/category statistics")
	completions := flag.String("completions", "", "generate shell completions (bash, zsh, fish)")
	completionsList := flag.Bool("completions-list", false, "list topics for shell completion (hidden)")
	interactive := flag.Bool("i", false, "interactive TUI mode")
	related := flag.String("related", "", "show related topics for a sheet")
	format := flag.String("format", "", "output format: markdown, json")
	star := flag.String("star", "", "toggle bookmark for a topic")
	starred := flag.Bool("starred", false, "list bookmarked topics")
	prereqs := flag.Bool("prereqs", false, "show prerequisites (use with -d)")
	update := flag.Bool("update", false, "check for updates and self-update")
	stackOverflow := flag.String("stack-overflow", "", "live Stack Overflow search (bonus opt-in; requires STACK_OVERFLOW_API_KEY — try '-so help')")

	// Short aliases
	flag.BoolVar(random, "r", false, "shorthand for -random")
	flag.BoolVar(count, "c", false, "shorthand for -count")
	flag.BoolVar(prereqs, "p", false, "shorthand for -prereqs")
	flag.BoolVar(update, "u", false, "shorthand for -update")
	flag.StringVar(add, "a", "", "shorthand for -add")
	flag.StringVar(edit, "e", "", "shorthand for -edit")
	flag.StringVar(related, "R", "", "shorthand for -related")
	flag.StringVar(format, "f", "", "shorthand for -format")
	flag.StringVar(star, "b", "", "shorthand for -star (bookmark)")
	flag.BoolVar(starred, "B", false, "shorthand for -starred")
	flag.StringVar(stackOverflow, "so", "", "shorthand for -stack-overflow")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `%[1]s - cheatsheet CLI (v%[2]s)

Usage:
  %[1]s                     list all topics grouped by category
  %[1]s <topic>             show cheatsheet (e.g., %[1]s lvm)
  %[1]s <category>          list topics in category (e.g., %[1]s storage)
  %[1]s <topic> <section>   show matching section (e.g., %[1]s lvm extend)
  %[1]s -d <topic>          deep theory/math for topic (e.g., %[1]s -d bgp)
  %[1]s -d <topic> --prereqs show prerequisites for a detail page
  %[1]s -s <term> [term...] AND-search across all cheatsheets (e.g., %[1]s -s python list)
  %[1]s -l                  list all topics with descriptions
  %[1]s -i                  interactive TUI mode (?: help, /: filter, t: theme)
  %[1]s --add <file>        add a custom cheatsheet
  %[1]s --edit <topic>      edit/create custom cheatsheet in $EDITOR

Tools:
  %[1]s calc <expression>   calculator (supports +,-,*,/,%%,**,hex,oct,bin,units)
  %[1]s subnet <cidr>       subnet calculator (e.g., %[1]s subnet 10.0.0.0/24)
  %[1]s compare <X> <Y>     compare two topics side by side
  %[1]s verify [topic]      verify math in detail pages
  %[1]s learn <category>    ordered learning path by prerequisites
  %[1]s serve               start REST API server (default :9876)

Knowledge:
  %[1]s --related <topic>   show related topics (from See Also)
  %[1]s --format json       export as JSON (use with topic)
  %[1]s --format markdown   export raw markdown (use with topic)

Bookmarks:
  %[1]s --star <topic>      toggle bookmark
  %[1]s --starred           list bookmarked topics

Extra:
  %[1]s --random            show a random cheatsheet
  %[1]s --count             show sheet/category statistics
  %[1]s --update            check for updates and self-update
  %[1]s --completions bash  generate shell completions (bash, zsh, fish)
  %[1]s -so help            Stack Overflow live lookup (bonus, opt-in)

Options:
`, prog, version)
		flag.PrintDefaults()
	}

	flag.Parse()

	if *ver {
		fmt.Printf("%s %s\n", prog, version)
		os.Exit(0)
	}

	if *completions != "" {
		doCompletions(*completions)
		return
	}

	if *update {
		doUpdate()
		return
	}

	// Build registry from embedded + custom sheets + detail sheets
	sheetSources := []fs.FS{}
	embedded, err := fs.Sub(vor.EmbeddedSheets, "sheets")
	if err != nil {
		die("embedded sheets: %v", err)
	}
	sheetSources = append(sheetSources, embedded)

	if customFS := custom.Load(); customFS != nil {
		sheetSources = append(sheetSources, customFS)
	}

	detailSources := []fs.FS{}
	detailFS, err := fs.Sub(vor.EmbeddedDetails, "detail")
	if err != nil {
		die("embedded details: %v", err)
	}
	detailSources = append(detailSources, detailFS)

	reg, err := registry.NewWithDetails(sheetSources, detailSources)
	if err != nil {
		die("load sheets: %v", err)
	}

	if *add != "" {
		// Optional category as the next positional arg:
		//   vor --add ./my-sheet.md networking
		// Without a category, the file lands in ~/.config/cs/sheets/uncategorized/
		// and the user is nudged toward existing custom categories on success.
		// --force allows overwriting an existing custom sheet at the destination.
		category := ""
		if extras := flag.Args(); len(extras) > 0 {
			category = extras[0]
		}
		if *force {
			custom.ConfirmOverwrite = true
		}
		if err := custom.AddTo(*add, category); err != nil {
			die("%v", err)
		}
		return
	}

	if *edit != "" {
		if err := custom.Edit(*edit, vor.EmbeddedSheets); err != nil {
			die("%v", err)
		}
		return
	}

	if *completionsList {
		doCompletionsList(reg)
		return
	}

	if *random {
		doRandom(reg)
		return
	}

	if *count {
		doCount(reg)
		return
	}

	if *starred {
		doStarred(reg)
		return
	}

	if *star != "" {
		doStar(*star, reg)
		return
	}

	if *related != "" {
		doRelated(reg, *related)
		return
	}

	if *interactive {
		doInteractive(reg)
		return
	}

	if *detail != "" {
		doDetail(reg, *detail, *prereqs, *format)
		return
	}

	if *stackOverflow != "" {
		stackoverflow.UserAgent = "vor-cli/" + version + " (https://github.com/bellistech/vor)"
		doStackOverflow(*stackOverflow)
		return
	}

	if *search != "" {
		terms := append([]string{*search}, flag.Args()...)
		doSearch(reg, terms)
		return
	}

	if *list {
		doList(reg)
		return
	}

	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "calc":
			if len(args) < 2 || isHelp(args[1:]) {
				calcHelp()
				return
			}
			doCalc(strings.Join(args[1:], " "))
			return
		case "subnet":
			if len(args) < 2 || isHelp(args[1:]) {
				subnetHelp()
				return
			}
			doSubnet(strings.Join(args[1:], " "))
			return
		case "compare":
			if len(args) < 3 {
				die("usage: cs compare <topic1> <topic2>")
			}
			doCompare(reg, args[1], args[2])
			return
		case "verify":
			if len(args) < 2 {
				doVerifyAll(reg)
			} else {
				doVerifyTopic(reg, args[1])
			}
			return
		case "learn":
			if len(args) < 2 {
				die("usage: cs learn <category>")
			}
			doLearn(reg, args[1])
			return
		case "serve":
			port := "9876"
			bind := "127.0.0.1"
			if len(args) > 1 {
				port = args[1]
			}
			doServe(reg, bind, port)
			return
		}
	}

	switch len(args) {
	case 0:
		doCategories(reg)
	case 1:
		doShow(reg, args[0], *format)
	default:
		doSection(reg, args[0], strings.Join(args[1:], " "), *format)
	}
}

func doCategories(reg *registry.Registry) {
	width := render.TermWidth()
	nameW := 22
	pad := 4 // leading indent + spacing
	descMax := width - nameW - pad
	if descMax < 20 {
		descMax = 20
	}

	var sb strings.Builder
	for _, cat := range reg.Categories() {
		sheets := reg.ByCategory(cat)
		sb.WriteString(fmt.Sprintf("\033[1;32m%s\033[0m\n\n", cat))
		for _, s := range sheets {
			desc := s.Description
			if len(desc) > descMax {
				desc = desc[:descMax-3] + "..."
			}
			sb.WriteString(fmt.Sprintf("  %-*s %s\n", nameW, s.Name, desc))
		}
		sb.WriteString("\n")
	}
	render.PlainOutput(sb.String())
}

func doList(reg *registry.Registry) {
	width := render.TermWidth()
	nameW := 22
	catW := 16
	pad := 4 // leading indent + spaces between columns
	descMax := width - nameW - catW - pad
	if descMax < 20 {
		descMax = 20
	}

	var sb strings.Builder
	sb.WriteString("\033[1;32mAll Cheatsheets\033[0m\n\n")
	for _, s := range reg.List() {
		desc := s.Description
		if len(desc) > descMax {
			desc = desc[:descMax-3] + "..."
		}
		cat := fmt.Sprintf("[%s]", s.Category)
		mark := "  "
		if bookmarks.IsBookmarked(s.Name) {
			mark = "* "
		}
		sb.WriteString(fmt.Sprintf("%s%-*s %-*s %s\n", mark, nameW, s.Name, catW, cat, desc))
	}
	render.PlainOutput(sb.String())
}

func doShow(reg *registry.Registry, name, format string) {
	// Check for exact sheet match first (takes priority over category)
	s := reg.Get(name)
	if s != nil {
		if format != "" {
			doExport(s, nil, format)
			return
		}
		render.Output(s.Content)
		// Show hints
		var hints []string
		if reg.HasDetail(s.Name) {
			hints = append(hints, fmt.Sprintf("cs -d %s", s.Name))
		}
		if len(s.SeeAlso) > 0 {
			hints = append(hints, fmt.Sprintf("cs --related %s", s.Name))
		}
		if len(hints) > 0 {
			fmt.Fprintf(os.Stderr, "\n\033[1;33mAlso try: %s\033[0m\n", strings.Join(hints, " | "))
		}
		return
	}

	// Then check if it's a category
	if reg.IsCategory(name) {
		width := render.TermWidth()
		nameW := 22
		pad := 4
		descMax := width - nameW - pad
		if descMax < 20 {
			descMax = 20
		}
		sheets := reg.ByCategory(name)
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("\033[1;32m%s\033[0m\n\n", name))
		for _, s := range sheets {
			desc := s.Description
			if len(desc) > descMax {
				desc = desc[:descMax-3] + "..."
			}
			sb.WriteString(fmt.Sprintf("  %-*s %s\n", nameW, s.Name, desc))
		}
		render.PlainOutput(sb.String())
		return
	}

	// Try fuzzy matching: prefix, substring, then Levenshtein
	lower := strings.ToLower(name)

	// Prefix match first (cs pg → postgresql)
	for _, sheet := range reg.List() {
		if strings.HasPrefix(sheet.Name, lower) {
			s = sheet
			break
		}
	}

	// Substring match
	if s == nil {
		for _, sheet := range reg.List() {
			if strings.Contains(sheet.Name, lower) {
				s = sheet
				break
			}
		}
	}

	// Levenshtein fuzzy match (cs kube → kubernetes)
	if s == nil {
		bestDist := len(name) + 1
		for _, sheet := range reg.List() {
			d := levenshtein(lower, sheet.Name)
			// Also try matching against prefix of sheet name
			if len(sheet.Name) > len(lower) {
				dp := levenshtein(lower, sheet.Name[:len(lower)])
				if dp < d {
					d = dp
				}
			}
			if d < bestDist && d <= len(name)/2+1 {
				bestDist = d
				s = sheet
			}
		}
	}

	if s == nil {
		die("unknown topic: %s (use 'cs -l' to list all)", name)
	}

	if format != "" {
		doExport(s, nil, format)
		return
	}
	render.Output(s.Content)
	var hints []string
	if reg.HasDetail(s.Name) {
		hints = append(hints, fmt.Sprintf("cs -d %s", s.Name))
	}
	if len(s.SeeAlso) > 0 {
		hints = append(hints, fmt.Sprintf("cs --related %s", s.Name))
	}
	if len(hints) > 0 {
		fmt.Fprintf(os.Stderr, "\n\033[1;33mAlso try: %s\033[0m\n", strings.Join(hints, " | "))
	}
}

func doDetail(reg *registry.Registry, name string, showPrereqs bool, format string) {
	d := reg.GetDetail(name)
	if d == nil {
		// Try fuzzy matching on detail names
		lower := strings.ToLower(name)
		for _, s := range reg.List() {
			if reg.HasDetail(s.Name) && strings.HasPrefix(s.Name, lower) {
				d = reg.GetDetail(s.Name)
				break
			}
		}
		if d == nil {
			for _, s := range reg.List() {
				if reg.HasDetail(s.Name) && strings.Contains(s.Name, lower) {
					d = reg.GetDetail(s.Name)
					break
				}
			}
		}
	}

	if d == nil {
		die("no detail available for: %s (use 'cs -l' to list all topics)", name)
	}

	if showPrereqs {
		if len(d.Prerequisites) > 0 {
			fmt.Printf("\n\033[1;32mPrerequisites for %s:\033[0m\n\n", d.Name)
			for _, p := range d.Prerequisites {
				fmt.Printf("  - %s\n", p)
			}
			fmt.Println()
		} else {
			fmt.Printf("  No prerequisites listed for %s.\n", d.Name)
		}
		return
	}

	if format != "" {
		doExport(nil, d, format)
		return
	}
	render.Output(d.Content)
}

func doSection(reg *registry.Registry, name, section, format string) {
	content, err := reg.FindSection(name, section)
	if err != nil {
		die("%v", err)
	}
	if format == "markdown" {
		fmt.Print(content)
		return
	}
	if format == "json" {
		data, _ := json.MarshalIndent(map[string]string{
			"topic":   name,
			"section": section,
			"content": content,
		}, "", "  ")
		fmt.Println(string(data))
		return
	}
	render.Output(content)
}

func doSearch(reg *registry.Registry, terms []string) {
	label := strings.Join(terms, " ")
	matches := reg.Search(terms...)
	if len(matches) == 0 {
		die("no results for: %s", label)
	}

	// Deduplicate by sheet+section
	type key struct{ sheet, section string }
	seen := make(map[key]bool)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Search: %s\n\n", label))

	count := 0
	for _, m := range matches {
		k := key{m.Sheet.Name, m.Section}
		if seen[k] {
			continue
		}
		seen[k] = true
		count++

		sec := m.Section
		if sec == "" {
			sec = "(top)"
		}
		sb.WriteString(fmt.Sprintf("  **%s/%s** :: %s\n", m.Sheet.Category, m.Sheet.Name, sec))
		sb.WriteString(fmt.Sprintf("    %s\n\n", m.Line))

		if count >= 25 {
			sb.WriteString(fmt.Sprintf("  ... and more. Use 'cs %s' to see full sheet.\n", matches[0].Sheet.Name))
			break
		}
	}
	render.Output(sb.String())
}

func doRelated(reg *registry.Registry, name string) {
	s := reg.Get(name)
	if s == nil {
		s = fuzzyFind(reg, name)
	}
	if s == nil {
		die("unknown topic: %s", name)
	}

	related := reg.Related(s.Name)
	if len(related) == 0 && len(s.SeeAlso) == 0 {
		fmt.Printf("  No related topics for %s.\n", s.Name)
		return
	}

	width := render.TermWidth()
	nameW := 22
	pad := 4
	descMax := width - nameW - pad
	if descMax < 20 {
		descMax = 20
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\033[1;32mRelated to %s:\033[0m\n\n", s.Name))
	for _, r := range related {
		desc := r.Description
		if len(desc) > descMax {
			desc = desc[:descMax-3] + "..."
		}
		detail := ""
		if reg.HasDetail(r.Name) {
			detail = " [detail]"
		}
		sb.WriteString(fmt.Sprintf("  %-*s %s%s\n", nameW, r.Name, desc, detail))
	}
	// Show unresolved references
	for _, ref := range s.SeeAlso {
		if reg.Get(ref) == nil {
			sb.WriteString(fmt.Sprintf("  %-*s (external)\n", nameW, ref))
		}
	}
	render.PlainOutput(sb.String())
}

func doCompare(reg *registry.Registry, nameA, nameB string) {
	a := reg.Get(nameA)
	if a == nil {
		a = fuzzyFind(reg, nameA)
	}
	b := reg.Get(nameB)
	if b == nil {
		b = fuzzyFind(reg, nameB)
	}
	if a == nil {
		die("unknown topic: %s", nameA)
	}
	if b == nil {
		die("unknown topic: %s", nameB)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Compare: %s vs %s\n\n", a.Name, b.Name))

	sb.WriteString("| Attribute | " + a.Name + " | " + b.Name + " |\n")
	sb.WriteString("|-----------|")
	sb.WriteString(strings.Repeat("-", len(a.Name)+2) + "|")
	sb.WriteString(strings.Repeat("-", len(b.Name)+2) + "|\n")

	sb.WriteString(fmt.Sprintf("| Category | %s | %s |\n", a.Category, b.Category))

	descA := truncate(a.Description, 40)
	descB := truncate(b.Description, 40)
	sb.WriteString(fmt.Sprintf("| Description | %s | %s |\n", descA, descB))

	sb.WriteString(fmt.Sprintf("| Sections | %d | %d |\n", len(a.Sections), len(b.Sections)))
	sb.WriteString(fmt.Sprintf("| Lines | %d | %d |\n",
		strings.Count(a.Content, "\n"), strings.Count(b.Content, "\n")))

	detailA := "No"
	detailB := "No"
	if reg.HasDetail(a.Name) {
		detailA = "Yes"
	}
	if reg.HasDetail(b.Name) {
		detailB = "Yes"
	}
	sb.WriteString(fmt.Sprintf("| Detail Page | %s | %s |\n", detailA, detailB))

	seeA := strings.Join(a.SeeAlso, ", ")
	seeB := strings.Join(b.SeeAlso, ", ")
	if seeA == "" {
		seeA = "-"
	}
	if seeB == "" {
		seeB = "-"
	}
	sb.WriteString(fmt.Sprintf("| See Also | %s | %s |\n", truncate(seeA, 30), truncate(seeB, 30)))

	sb.WriteString("\n## Sections\n\n")
	// Show section comparison
	secA := sectionNames(a)
	secB := sectionNames(b)
	allSec := mergeKeys(secA, secB)
	sb.WriteString(fmt.Sprintf("| Section | %s | %s |\n", a.Name, b.Name))
	sb.WriteString("|---------|:---:|:---:|\n")
	for _, sec := range allSec {
		inA := " "
		inB := " "
		if secA[sec] {
			inA = "Y"
		}
		if secB[sec] {
			inB = "Y"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", sec, inA, inB))
	}

	render.Output(sb.String())
}

func doVerifyAll(reg *registry.Registry) {
	totalPass := 0
	totalFail := 0
	totalTopics := 0

	for _, s := range reg.List() {
		d := reg.GetDetail(s.Name)
		if d == nil {
			continue
		}
		r := verify.Verify(d.Name, d.Content)
		if len(r.Results) == 0 {
			continue
		}
		totalTopics++
		totalPass += r.Pass
		totalFail += r.Fail
		if r.Fail > 0 {
			fmt.Print(verify.Format(r))
		}
	}
	fmt.Printf("\n\033[1;32mVerification Summary\033[0m\n")
	fmt.Printf("  Topics checked: %d\n", totalTopics)
	fmt.Printf("  Expressions:    %d pass, %d fail\n", totalPass, totalFail)
	if totalFail > 0 {
		os.Exit(1)
	}
}

func doVerifyTopic(reg *registry.Registry, name string) {
	d := reg.GetDetail(name)
	if d == nil {
		die("no detail available for: %s", name)
	}
	r := verify.Verify(d.Name, d.Content)
	fmt.Print(verify.Format(r))
	if r.Fail > 0 {
		os.Exit(1)
	}
}

func doExport(sheet *registry.Sheet, detail *registry.Sheet, format string) {
	s := sheet
	if s == nil {
		s = detail
	}
	switch format {
	case "markdown":
		fmt.Print(s.Content)
	case "json":
		type jsonSheet struct {
			Name          string   `json:"name"`
			Category      string   `json:"category"`
			Title         string   `json:"title"`
			Description   string   `json:"description"`
			Content       string   `json:"content"`
			SeeAlso       []string `json:"see_also,omitempty"`
			Prerequisites []string `json:"prerequisites,omitempty"`
			Complexity    string   `json:"complexity,omitempty"`
			Sections      []struct {
				Title   string `json:"title"`
				Level   int    `json:"level"`
				Content string `json:"content"`
			} `json:"sections"`
		}
		j := jsonSheet{
			Name:          s.Name,
			Category:      s.Category,
			Title:         s.Title,
			Description:   s.Description,
			Content:       s.Content,
			SeeAlso:       s.SeeAlso,
			Prerequisites: s.Prerequisites,
			Complexity:    s.Complexity,
		}
		for _, sec := range s.Sections {
			j.Sections = append(j.Sections, struct {
				Title   string `json:"title"`
				Level   int    `json:"level"`
				Content string `json:"content"`
			}{sec.Title, sec.Level, sec.Content})
		}
		data, _ := json.MarshalIndent(j, "", "  ")
		fmt.Println(string(data))
	default:
		die("unknown format: %s (use 'markdown' or 'json')", format)
	}
}

func doStar(name string, reg *registry.Registry) {
	s := reg.Get(name)
	if s == nil {
		s = fuzzyFind(reg, name)
	}
	if s == nil {
		die("unknown topic: %s", name)
	}

	added, err := bookmarks.Toggle(s.Name)
	if err != nil {
		die("bookmark: %v", err)
	}
	if added {
		fmt.Printf("  Bookmarked: %s\n", s.Name)
	} else {
		fmt.Printf("  Removed bookmark: %s\n", s.Name)
	}
}

func doStarred(reg *registry.Registry) {
	marks := bookmarks.List()
	if len(marks) == 0 {
		fmt.Println("  No bookmarks. Use 'cs --star <topic>' to add one.")
		return
	}

	width := render.TermWidth()
	nameW := 22
	pad := 4
	descMax := width - nameW - pad
	if descMax < 20 {
		descMax = 20
	}

	var sb strings.Builder
	sb.WriteString("\033[1;32mBookmarked Topics\033[0m\n\n")
	for _, name := range marks {
		s := reg.Get(name)
		if s == nil {
			sb.WriteString(fmt.Sprintf("  %-*s (not found)\n", nameW, name))
			continue
		}
		desc := s.Description
		if len(desc) > descMax {
			desc = desc[:descMax-3] + "..."
		}
		sb.WriteString(fmt.Sprintf("  %-*s %s\n", nameW, s.Name, desc))
	}
	render.PlainOutput(sb.String())
}

func doLearn(reg *registry.Registry, catName string) {
	if !reg.IsCategory(catName) {
		die("unknown category: %s", catName)
	}

	sheets := reg.ByCategory(catName)
	if len(sheets) == 0 {
		die("no sheets in category: %s", catName)
	}

	// Build dependency graph from detail prerequisites
	// Topics with fewer prerequisites come first (topological-ish sort)
	type entry struct {
		sheet       *registry.Sheet
		prereqCount int
	}
	var entries []entry
	for _, s := range sheets {
		d := reg.GetDetail(s.Name)
		pCount := 0
		if d != nil {
			pCount = len(d.Prerequisites)
		}
		entries = append(entries, entry{s, pCount})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].prereqCount != entries[j].prereqCount {
			return entries[i].prereqCount < entries[j].prereqCount
		}
		return entries[i].sheet.Name < entries[j].sheet.Name
	})

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\033[1;32mLearning Path: %s\033[0m\n\n", catName))
	for i, e := range entries {
		detail := ""
		if reg.HasDetail(e.sheet.Name) {
			detail = " [has detail]"
		}
		prereq := ""
		d := reg.GetDetail(e.sheet.Name)
		if d != nil && len(d.Prerequisites) > 0 {
			prereq = fmt.Sprintf(" (prereqs: %s)", strings.Join(d.Prerequisites, ", "))
		}
		sb.WriteString(fmt.Sprintf("  %2d. %-20s %s%s%s\n", i+1, e.sheet.Name, truncate(e.sheet.Description, 40), detail, prereq))
	}
	render.PlainOutput(sb.String())
}

func doUpdate() {
	fmt.Printf("%s %s (%s/%s)\n", progName(), version, runtime.GOOS, runtime.GOARCH)
	fmt.Println("Checking for updates...")

	resp, err := http.Get("https://api.github.com/repos/bellistech/vor/releases/latest")
	if err != nil {
		die("check update: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		die("GitHub API returned %d (no releases found?)", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		die("parse release: %v", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(version, "v")
	if latest == current {
		fmt.Printf("Already up to date (v%s).\n", current)
		return
	}

	fmt.Printf("Update available: %s → %s\n", current, latest)
	fmt.Printf("Download: %s\n", release.HTMLURL)

	// Look for matching binary
	target := fmt.Sprintf("cs-%s-%s", runtime.GOOS, runtime.GOARCH)
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, target) {
			fmt.Printf("Binary: %s\n", asset.BrowserDownloadURL)

			// Download and replace
			binResp, err := http.Get(asset.BrowserDownloadURL)
			if err != nil {
				die("download: %v", err)
			}
			defer binResp.Body.Close()

			exe, err := os.Executable()
			if err != nil {
				die("find executable: %v", err)
			}

			tmpFile := exe + ".new"
			f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				die("create temp: %v", err)
			}
			if _, err := io.Copy(f, binResp.Body); err != nil {
				f.Close()
				os.Remove(tmpFile)
				die("download: %v", err)
			}
			f.Close()

			if err := os.Rename(tmpFile, exe); err != nil {
				os.Remove(tmpFile)
				die("replace binary: %v", err)
			}
			fmt.Printf("Updated to %s!\n", latest)
			return
		}
	}

	fmt.Println("No matching binary found for your platform.")
	fmt.Printf("Build from source: %s\n", release.HTMLURL)
}

func doInteractive(reg *registry.Registry) {
	if err := tui.Run(reg); err != nil {
		die("tui: %v", err)
	}
}

// doServe starts a REST API server
func doServe(reg *registry.Registry, bind, port string) {
	mux := http.NewServeMux()

	// CORS middleware
	cors := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")
			if r.Method == "OPTIONS" {
				w.WriteHeader(200)
				return
			}
			next(w, r)
		}
	}

	jsonResp := func(w http.ResponseWriter, v any) {
		data, _ := json.MarshalIndent(v, "", "  ")
		w.Write(data)
	}

	// GET /api/version — build identification, useful for clients to verify
	// they're talking to the version they expect. No auth required.
	mux.HandleFunc("/api/version", cors(func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, map[string]any{
			"binary":     "vor",
			"version":    version,
			"goos":       runtime.GOOS,
			"goarch":     runtime.GOARCH,
			"go_version": runtime.Version(),
		})
	}))

	// GET /api/health — liveness probe; cheap counters only, no work.
	// Returns 200 with sheet counts if the registry loaded; useful for
	// container orchestrators (k8s livenessProbe, docker HEALTHCHECK, etc).
	mux.HandleFunc("/api/health", cors(func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, map[string]any{
			"status":     "ok",
			"sheets":     len(reg.List()),
			"categories": len(reg.Categories()),
			"details":    reg.DetailCount(),
		})
	}))

	// GET /api/openapi — minimal OpenAPI 3.0 hint listing the endpoints.
	// Not a complete spec; just enough to let api-explorer tools discover.
	mux.HandleFunc("/api/openapi", cors(func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, openAPIDoc(version))
	}))

	// GET /api/topics
	mux.HandleFunc("/api/topics", cors(func(w http.ResponseWriter, r *http.Request) {
		type topicSummary struct {
			Name        string   `json:"name"`
			Category    string   `json:"category"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			HasDetail   bool     `json:"has_detail"`
			SeeAlso     []string `json:"see_also,omitempty"`
		}
		var topics []topicSummary
		for _, s := range reg.List() {
			topics = append(topics, topicSummary{
				Name:        s.Name,
				Category:    s.Category,
				Title:       s.Title,
				Description: s.Description,
				HasDetail:   reg.HasDetail(s.Name),
				SeeAlso:     s.SeeAlso,
			})
		}
		jsonResp(w, topics)
	}))

	// GET /api/topics/<name>
	mux.HandleFunc("/api/topics/", cors(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/topics/")
		parts := strings.SplitN(path, "/", 2)
		name := parts[0]

		s := reg.Get(name)
		if s == nil {
			s = fuzzyFind(reg, name)
		}
		if s == nil {
			http.Error(w, `{"error":"not found"}`, 404)
			return
		}

		// Check for sub-routes
		if len(parts) > 1 {
			switch parts[1] {
			case "detail":
				d := reg.GetDetail(s.Name)
				if d == nil {
					http.Error(w, `{"error":"no detail"}`, 404)
					return
				}
				jsonResp(w, map[string]any{
					"name":          d.Name,
					"category":      d.Category,
					"content":       d.Content,
					"prerequisites": d.Prerequisites,
					"complexity":    d.Complexity,
				})
				return
			case "related":
				related := reg.Related(s.Name)
				var names []string
				for _, r := range related {
					names = append(names, r.Name)
				}
				jsonResp(w, map[string]any{"topic": s.Name, "related": names})
				return
			}
		}

		jsonResp(w, map[string]any{
			"name":        s.Name,
			"category":    s.Category,
			"title":       s.Title,
			"description": s.Description,
			"content":     s.Content,
			"see_also":    s.SeeAlso,
			"has_detail":  reg.HasDetail(s.Name),
		})
	}))

	// GET /api/categories
	mux.HandleFunc("/api/categories", cors(func(w http.ResponseWriter, r *http.Request) {
		type catInfo struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}
		var cats []catInfo
		for _, c := range reg.Categories() {
			cats = append(cats, catInfo{c, len(reg.ByCategory(c))})
		}
		jsonResp(w, cats)
	}))

	// GET /api/search?q=<query>
	mux.HandleFunc("/api/search", cors(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q == "" {
			http.Error(w, `{"error":"missing q parameter"}`, 400)
			return
		}
		matches := reg.Search(q)
		type matchResult struct {
			Topic   string `json:"topic"`
			Category string `json:"category"`
			Section string `json:"section"`
			Line    string `json:"line"`
		}
		var results []matchResult
		seen := make(map[string]bool)
		for _, m := range matches {
			key := m.Sheet.Name + "|" + m.Section
			if seen[key] {
				continue
			}
			seen[key] = true
			results = append(results, matchResult{
				Topic:    m.Sheet.Name,
				Category: m.Sheet.Category,
				Section:  m.Section,
				Line:     m.Line,
			})
			if len(results) >= 50 {
				break
			}
		}
		jsonResp(w, results)
	}))

	// GET /api/compare?a=<topic>&b=<topic>
	mux.HandleFunc("/api/compare", cors(func(w http.ResponseWriter, r *http.Request) {
		a := reg.Get(r.URL.Query().Get("a"))
		b := reg.Get(r.URL.Query().Get("b"))
		if a == nil || b == nil {
			http.Error(w, `{"error":"both topics required"}`, 400)
			return
		}
		jsonResp(w, map[string]any{
			"a": map[string]any{
				"name": a.Name, "category": a.Category, "sections": len(a.Sections),
				"lines": strings.Count(a.Content, "\n"), "see_also": a.SeeAlso,
			},
			"b": map[string]any{
				"name": b.Name, "category": b.Category, "sections": len(b.Sections),
				"lines": strings.Count(b.Content, "\n"), "see_also": b.SeeAlso,
			},
		})
	}))

	// POST /api/calc
	mux.HandleFunc("/api/calc", cors(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, `{"error":"POST required"}`, 405)
			return
		}
		var req struct {
			Expr string `json:"expr"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Expr == "" {
			http.Error(w, `{"error":"missing expr"}`, 400)
			return
		}
		result, err := calc.Eval(req.Expr)
		if err != nil {
			jsonResp(w, map[string]any{"error": err.Error()})
			return
		}
		jsonResp(w, map[string]any{"expr": req.Expr, "result": result})
	}))

	// POST /api/subnet
	mux.HandleFunc("/api/subnet", cors(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, `{"error":"POST required"}`, 405)
			return
		}
		var req struct {
			CIDR string `json:"cidr"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.CIDR == "" {
			http.Error(w, `{"error":"missing cidr"}`, 400)
			return
		}
		info, err := subnet.Calculate(req.CIDR)
		if err != nil {
			jsonResp(w, map[string]any{"error": err.Error()})
			return
		}
		jsonResp(w, map[string]any{
			"cidr":        info.CIDR,
			"network":     info.Network.String(),
			"prefix":      info.Prefix,
			"first_host":  info.FirstHost.String(),
			"last_host":   info.LastHost.String(),
			"total_hosts": info.TotalHosts.String(),
			"usable_hosts": info.UsableHosts.String(),
			"is_ipv6":     info.IsIPv6,
		})
	}))

	// GET /api/verify/<name>
	mux.HandleFunc("/api/verify/", cors(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/verify/")
		d := reg.GetDetail(name)
		if d == nil {
			http.Error(w, `{"error":"no detail"}`, 404)
			return
		}
		report := verify.Verify(d.Name, d.Content)
		jsonResp(w, report)
	}))

	// GET /api/stats
	mux.HandleFunc("/api/stats", cors(func(w http.ResponseWriter, r *http.Request) {
		sheets := reg.List()
		totalLines := 0
		for _, s := range sheets {
			totalLines += strings.Count(s.Content, "\n")
		}
		jsonResp(w, map[string]any{
			"sheets":       len(sheets),
			"details":      reg.DetailCount(),
			"categories":   len(reg.Categories()),
			"total_lines":  totalLines,
			"see_also":     reg.SeeAlsoCoverage(),
			"bookmarks":    len(bookmarks.List()),
		})
	}))

	// GET /api/bookmarks + POST /api/bookmarks/<name>
	mux.HandleFunc("/api/bookmarks", cors(func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, bookmarks.List())
	}))
	mux.HandleFunc("/api/bookmarks/", cors(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/bookmarks/")
		if r.Method != "POST" {
			http.Error(w, `{"error":"POST required"}`, 405)
			return
		}
		added, err := bookmarks.Toggle(name)
		if err != nil {
			jsonResp(w, map[string]any{"error": err.Error()})
			return
		}
		jsonResp(w, map[string]any{"topic": name, "bookmarked": added})
	}))

	// GET /api/stackoverflow?q=<query> — bonus opt-in. Server-side reads
	// STACK_OVERFLOW_API_KEY from env or ~/.config/cs/secrets.env. If no
	// key is configured, returns 503 Service Unavailable so callers can
	// surface a friendly setup nudge without conflating with auth failure.
	mux.HandleFunc("/api/stackoverflow", cors(func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			http.Error(w, `{"error":"missing q parameter"}`, 400)
			return
		}
		if cached, hit := stackoverflow.Read(query, 24*time.Hour); hit {
			jsonResp(w, cached)
			return
		}
		key, _, err := secrets.Load("STACK_OVERFLOW_API_KEY")
		if err != nil {
			http.Error(w, `{"error":"STACK_OVERFLOW_API_KEY not configured (bonus opt-in feature)"}`, 503)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		res, err := stackoverflow.Search(ctx, query, key)
		if err != nil {
			status := 502
			switch {
			case errors.Is(err, stackoverflow.ErrAuth):
				status = 401
			case errors.Is(err, stackoverflow.ErrRateLimited):
				status = 429
			case errors.Is(err, stackoverflow.ErrEmptyQuery):
				status = 400
			}
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), status)
			return
		}
		_ = stackoverflow.Write(query, res) // best-effort
		jsonResp(w, res)
	}))

	addr := bind + ":" + port
	fmt.Printf("\033[1;32mcs serve\033[0m listening on %s\n", addr)
	fmt.Println("Endpoints:")
	fmt.Println("  GET  /api/version             — build identification")
	fmt.Println("  GET  /api/health              — liveness probe")
	fmt.Println("  GET  /api/openapi             — OpenAPI 3.0 endpoint hint")
	fmt.Println("  GET  /api/topics              — list all topics")
	fmt.Println("  GET  /api/topics/:name        — get sheet")
	fmt.Println("  GET  /api/topics/:name/detail — get detail page")
	fmt.Println("  GET  /api/topics/:name/related— get related topics")
	fmt.Println("  GET  /api/categories          — list categories")
	fmt.Println("  GET  /api/search?q=           — search sheets")
	fmt.Println("  GET  /api/compare?a=&b=       — compare topics")
	fmt.Println("  POST /api/calc                — evaluate expression")
	fmt.Println("  POST /api/subnet              — subnet calculator")
	fmt.Println("  GET  /api/verify/:name        — verify detail math")
	fmt.Println("  GET  /api/stats               — statistics")
	fmt.Println("  GET  /api/bookmarks           — list bookmarks")
	fmt.Println("  POST /api/bookmarks/:name     — toggle bookmark")
	fmt.Println("  GET  /api/stackoverflow?q=    — Stack Overflow lookup (bonus, opt-in)")
	fmt.Println()
	if err := http.ListenAndServe(addr, mux); err != nil {
		die("serve: %v", err)
	}
}

func isHelp(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "-h", "--help", "help":
		return true
	}
	return false
}

func calcHelp() {
	help := `# cs calc — Calculator

Evaluate arithmetic expressions with base conversion and unit support.

## Usage

    cs calc <expression>

## Operators

    +   addition            cs calc "10 + 3"       → 13
    -   subtraction         cs calc "10 - 3"       → 7
    *   multiplication      cs calc "4 * 5"        → 20
    /   division            cs calc "15 / 4"       → 3.75
    %   modulo              cs calc "17 % 5"       → 2
    **  power               cs calc "2 ** 10"      → 1024
    ()  grouping            cs calc "(2 + 3) * 4"  → 20

## Number Formats

    decimal     cs calc "255"          → 255
    hex         cs calc "0xFF"         → 255
    octal       cs calc "0o377"        → 255
    binary      cs calc "0b11111111"   → 255
    float       cs calc "3.14 * 2"    → 6.28
    scientific  cs calc "1e6 + 1"     → 1000001

## Units

    cs calc "10GB / 2"         → 5 GB
    cs calc "10Gbps / 8"       → 1.25 Gbps
    cs calc "500ms * 2"        → 1 s
    cs calc "1GiB"             → 1,073,741,824 B

    Data:  B, KB, MB, GB, TB, PB, KiB, MiB, GiB, TiB
    Rate:  bps, Kbps, Mbps, Gbps, Tbps
    Time:  ns, us, ms, s, min, hr

## Functions

    sqrt(x)     square root       cs calc "sqrt(144)"     → 12
    abs(x)      absolute value    cs calc "abs(-42)"      → 42
    log(x)      log base 10      cs calc "log(1000)"     → 3
    ln(x)       natural log       cs calc "ln(e)"         → 1
    log2(x)     log base 2        cs calc "log2(1024)"    → 10
    ceil(x)     ceiling           cs calc "ceil(3.2)"     → 4
    floor(x)    floor             cs calc "floor(3.9)"    → 3

## Constants

    pi          3.14159...        cs calc "pi * 10 ** 2"  → 314.159...
    e           2.71828...        cs calc "e ** 2"         → 7.389...

## Output

Integer results include base conversions:

    cs calc "0xFF"
      = 255
      hex  0xFF
      oct  0o377
      bin  0b11111111

## Examples

    cs calc "2 ** 16"                 → 65536
    cs calc "0xFF + 0b1010"          → 265
    cs calc "sqrt(2) * 100"          → 141.421...
    cs calc "(1024 * 1024) / 8"      → 131072
    cs calc "0xDEADBEEF"             → 3735928559
`
	render.Output(help)
}

func subnetHelp() {
	help := `# cs subnet — Subnet Calculator

Calculate network, broadcast, host range, and mask details for IPv4/IPv6 subnets.

## Usage

    cs subnet <cidr>
    cs subnet <ip> <mask>

## Input Formats

    CIDR notation:     cs subnet 192.168.1.0/24
    IP + mask:         cs subnet 192.168.1.0 255.255.255.0
    Any host in net:   cs subnet 192.168.1.47/24
    IPv6:              cs subnet 2001:db8::/32

## Output Fields (IPv4)

    Network          network address
    Broadcast        broadcast address
    Netmask          dotted-decimal mask
    Wildcard         inverse mask (for ACLs)
    Prefix           CIDR prefix length
    First Host       first usable host address
    Last Host        last usable host address
    Total Addrs      total addresses in subnet (2^host bits)
    Usable Hosts     total - 2 (network + broadcast)
    Mask (binary)    binary representation of the netmask
    Class            classful network class (A/B/C/D/E)
    Type             Private (RFC 1918), Public, Loopback, etc.

## Output Fields (IPv6)

    Network, Prefix, First/Last Addr, Total Addrs, Type
    /64 Subnets      number of /64 subnets (if prefix < 64)

## Examples

    cs subnet 10.0.0.0/8              → 16,777,214 usable hosts
    cs subnet 172.16.0.0/22           → 4 x /24 blocks (1,022 hosts)
    cs subnet 192.168.1.0/28          → 14 usable hosts
    cs subnet 192.168.1.0/30          → point-to-point (2 hosts)
    cs subnet 192.168.1.0/31          → RFC 3021 point-to-point
    cs subnet 2001:db8:abcd::/48      → 65,536 /64 subnets

## Quick Reference

    /8    255.0.0.0         16,777,214 hosts
    /16   255.255.0.0       65,534 hosts
    /20   255.255.240.0     4,094 hosts
    /22   255.255.252.0     1,022 hosts
    /24   255.255.255.0     254 hosts
    /25   255.255.255.128   126 hosts
    /26   255.255.255.192   62 hosts
    /27   255.255.255.224   30 hosts
    /28   255.255.255.240   14 hosts
    /29   255.255.255.248   6 hosts
    /30   255.255.255.252   2 hosts
    /31   255.255.255.254   2 (point-to-point)
    /32   255.255.255.255   1 (host route)
`
	render.Output(help)
}

func doRandom(reg *registry.Registry) {
	sheets := reg.List()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	s := sheets[r.Intn(len(sheets))]
	fmt.Fprintf(os.Stderr, "\033[1;33mRandom: %s [%s]\033[0m\n\n", s.Name, s.Category)
	render.Output(s.Content)
}

func doCount(reg *registry.Registry) {
	sheets := reg.List()
	cats := reg.Categories()

	fmt.Printf("\n\033[1;32mcs statistics\033[0m\n\n")
	fmt.Printf("  %-20s %d\n", "Total sheets:", len(sheets))
	fmt.Printf("  %-20s %d/%d\n", "Detail pages:", reg.DetailCount(), len(sheets))
	fmt.Printf("  %-20s %d\n", "Categories:", len(cats))
	fmt.Printf("  %-20s %d/%d\n", "See Also coverage:", reg.SeeAlsoCoverage(), len(sheets))
	fmt.Printf("  %-20s %d\n", "Bookmarks:", len(bookmarks.List()))

	totalLines := 0
	for _, s := range sheets {
		totalLines += strings.Count(s.Content, "\n")
	}
	fmt.Printf("  %-20s %d\n", "Total lines:", totalLines)
	fmt.Println()

	// Per-category counts, sorted
	type catCount struct {
		name  string
		count int
	}
	var counts []catCount
	for _, cat := range cats {
		counts = append(counts, catCount{cat, len(reg.ByCategory(cat))})
	}
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	fmt.Printf("  \033[1mSheets per category:\033[0m\n")
	for _, c := range counts {
		bar := strings.Repeat("█", c.count)
		fmt.Printf("  %-20s %2d %s\n", c.name, c.count, bar)
	}
	fmt.Println()
}

func doCompletions(shell string) {
	// Use the actual command name (e.g. "vor" or "cs" via symlink) so completions
	// are correct for whichever alias the user invoked.
	name := filepath.Base(os.Args[0])
	if name == "" || name == "." || name == "/" {
		name = "vor"
	}
	var tmpl string
	switch shell {
	case "bash":
		tmpl = bashCompletion
	case "zsh":
		tmpl = zshCompletion
	case "fish":
		tmpl = fishCompletion
	default:
		die("unknown shell: %s (supported: bash, zsh, fish)", shell)
	}
	fmt.Print(strings.ReplaceAll(tmpl, "{{NAME}}", name))
}

func doCompletionsList(reg *registry.Registry) {
	// Print topics, categories, and subcommands for dynamic completion
	seen := make(map[string]bool)
	for _, s := range reg.List() {
		if !seen[s.Name] {
			fmt.Println(s.Name)
			seen[s.Name] = true
		}
	}
	for _, cat := range reg.Categories() {
		if !seen[cat] {
			fmt.Println(cat)
			seen[cat] = true
		}
	}
	fmt.Println("calc")
	fmt.Println("subnet")
	fmt.Println("compare")
	fmt.Println("verify")
	fmt.Println("learn")
	fmt.Println("serve")
}

func doCalc(expr string) {
	// Try unit-aware first
	unitResult, err := calc.EvalWithUnits(expr)
	if err == nil && unitResult.Unit != "" {
		fmt.Printf("\n\033[1;36m%s\033[0m\n\n", expr)
		fmt.Println(calc.FormatWithUnit(unitResult))
		fmt.Println()
		return
	}

	// Fall back to standard calc
	result, err := calc.Eval(expr)
	if err != nil {
		die("calc: %v", err)
	}
	fmt.Printf("\n\033[1;36m%s\033[0m\n\n", expr)
	fmt.Println(calc.Format(result))
	fmt.Println()
}

func doSubnet(input string) {
	info, err := subnet.Calculate(input)
	if err != nil {
		die("subnet: %v", err)
	}
	fmt.Println()
	fmt.Print(subnet.Format(info))
	fmt.Println()
}

// fuzzyFind attempts prefix, substring, then Levenshtein matching.
func fuzzyFind(reg *registry.Registry, name string) *registry.Sheet {
	lower := strings.ToLower(name)

	for _, sheet := range reg.List() {
		if strings.HasPrefix(sheet.Name, lower) {
			return sheet
		}
	}
	for _, sheet := range reg.List() {
		if strings.Contains(sheet.Name, lower) {
			return sheet
		}
	}

	var best *registry.Sheet
	bestDist := len(name) + 1
	for _, sheet := range reg.List() {
		d := levenshtein(lower, sheet.Name)
		if len(sheet.Name) > len(lower) {
			dp := levenshtein(lower, sheet.Name[:len(lower)])
			if dp < d {
				d = dp
			}
		}
		if d < bestDist && d <= len(name)/2+1 {
			bestDist = d
			best = sheet
		}
	}
	return best
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func sectionNames(s *registry.Sheet) map[string]bool {
	m := make(map[string]bool)
	for _, sec := range s.Sections {
		if sec.Level == 2 {
			m[sec.Title] = true
		}
	}
	return m
}

func mergeKeys(a, b map[string]bool) []string {
	all := make(map[string]bool)
	for k := range a {
		all[k] = true
	}
	for k := range b {
		all[k] = true
	}
	var keys []string
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const bashCompletion = `# cs bash completion
_{{NAME}}() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"

    case "$prev" in
        -s|--add|--edit|--star|--related|-d|--format)
            return 0
            ;;
        --completions)
            COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur"))
            return 0
            ;;
        compare)
            local topics
            topics=$({{NAME}} --completions-list 2>/dev/null)
            COMPREPLY=($(compgen -W "$topics" -- "$cur"))
            return 0
            ;;
    esac

    if [[ "$cur" == -* ]]; then
        COMPREPLY=($(compgen -W "-s -l -v -h -d -i --add --edit --random --count --completions --related --format --star --starred --prereqs --update --help" -- "$cur"))
        return 0
    fi

    local topics
    topics=$({{NAME}} --completions-list 2>/dev/null)
    COMPREPLY=($(compgen -W "$topics" -- "$cur"))
}
complete -F _{{NAME}} {{NAME}}
`

const zshCompletion = `#compdef {{NAME}}
_{{NAME}}() {
    local -a topics flags subcommands

    flags=(
        '-s[search across all cheatsheets]:query:'
        '-l[list all topics with descriptions]'
        '-v[print version]'
        '-h[show help]'
        '-d[show deep theory/math for topic]:topic:'
        '-i[interactive TUI mode]'
        '--add[add custom cheatsheet from file]:file:_files'
        '--edit[edit/create custom cheatsheet in EDITOR]:topic:'
        '--random[show a random cheatsheet]'
        '--count[show sheet/category statistics]'
        '--completions[generate shell completions]:shell:(bash zsh fish)'
        '--related[show related topics]:topic:'
        '--format[output format]:format:(markdown json)'
        '--star[toggle bookmark]:topic:'
        '--starred[list bookmarked topics]'
        '--prereqs[show prerequisites]'
        '--update[check for updates]'
    )

    if (( CURRENT == 2 )); then
        topics=("${(@f)$({{NAME}} --completions-list 2>/dev/null)}")
        _describe 'topics' topics -- || _arguments $flags
    elif (( CURRENT == 3 )); then
        return 0
    fi
}

_{{NAME}} "$@"
`

const fishCompletion = `# cs fish completion
complete -c {{NAME}} -f

# Flags
complete -c {{NAME}} -s s -l search -d "Search across all cheatsheets" -r
complete -c {{NAME}} -s l -d "List all topics with descriptions"
complete -c {{NAME}} -s v -d "Print version"
complete -c {{NAME}} -s h -d "Show help"
complete -c {{NAME}} -s d -d "Show deep theory/math for topic" -r
complete -c {{NAME}} -s i -d "Interactive TUI mode"
complete -c {{NAME}} -l add -d "Add custom cheatsheet from file" -r -F
complete -c {{NAME}} -l edit -d "Edit/create custom cheatsheet" -r
complete -c {{NAME}} -l random -d "Show a random cheatsheet"
complete -c {{NAME}} -l count -d "Show statistics"
complete -c {{NAME}} -l completions -d "Generate shell completions" -r -a "bash zsh fish"
complete -c {{NAME}} -l related -d "Show related topics" -r
complete -c {{NAME}} -l format -d "Output format" -r -a "markdown json"
complete -c {{NAME}} -l star -d "Toggle bookmark" -r
complete -c {{NAME}} -l starred -d "List bookmarked topics"
complete -c {{NAME}} -l prereqs -d "Show prerequisites"
complete -c {{NAME}} -l update -d "Check for updates"

# Topics and categories (dynamic)
complete -c {{NAME}} -n "not __fish_seen_subcommand_from ({{NAME}} --completions-list 2>/dev/null)" \
    -a "({{NAME}} --completions-list 2>/dev/null)"
`

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, progName()+": "+format+"\n", args...)
	os.Exit(1)
}

// doStackOverflow runs the optional, opt-in Stack Overflow live search. The
// offline encyclopedia (the heart of vör) does not invoke this path. It is
// gated on STACK_OVERFLOW_API_KEY being configured (env or
// ~/.config/cs/secrets.env). Without a key, prints friendly onboarding text
// and exits with status 1.
func doStackOverflow(query string) {
	q := strings.TrimSpace(query)
	if q == "" || q == "help" || q == "-h" || q == "--help" {
		stackOverflowHelp()
		return
	}

	// Try the cache first (treats stale entries as miss; corrupt files as miss).
	if cached, hit := stackoverflow.Read(q, 24*time.Hour); hit {
		if err := render.Output(stackoverflow.ToMarkdown(cached, q)); err != nil {
			die("render: %v", err)
		}
		return
	}

	// Load the key. Env wins; file falls back. Neither set → friendly nudge.
	key, _, err := secrets.Load("STACK_OVERFLOW_API_KEY")
	if err != nil {
		if errors.Is(err, secrets.ErrNotSet) {
			fmt.Fprintln(os.Stderr,
				"cs: -so requires STACK_OVERFLOW_API_KEY (env var or "+
					secrets.File()+"). Run 'cs -so help' for setup.")
			os.Exit(1)
		}
		die("secrets: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	res, err := stackoverflow.Search(ctx, q, key)
	if err != nil {
		die("stack-overflow: %v", err)
	}

	// Best-effort cache write. Failure is non-fatal — we already have the
	// result for this run; cache is a future-call optimization only.
	if werr := stackoverflow.Write(q, res); werr != nil {
		fmt.Fprintf(os.Stderr, "warning: stack-overflow cache write failed: %v\n", werr)
	}

	if err := render.Output(stackoverflow.ToMarkdown(res, q)); err != nil {
		die("render: %v", err)
	}
}

// stackOverflowHelp prints the onboarding block for the bonus opt-in feature.
// Reachable via `vor -so help` or `vor --stack-overflow help`.
func stackOverflowHelp() {
	fmt.Print(`Stack Overflow live lookup (BONUS — optional, off by default)
=================================================================

vör's primary feature is the offline encyclopedia. The default 'vor' usage
makes zero network calls. The '-so' flag is an opt-in shortcut for the
ad-hoc 5% of cases the offline corpus doesn't cover (fresh error messages,
version-specific gotchas).

To enable:

  1. Sign in at https://stackapps.com/users/login (any Stack Exchange
     account works — Stack Overflow login is fine).
  2. Register an app: https://stackapps.com/apps/oauth/register
       Application Name:    vor-cli (any name)
       OAuth Domain:        localhost  (literal — we don't use OAuth)
       Application Website: any URL you control
       Enable Client-side OAuth Flow: NO (leave unchecked)
  3. Copy the 'Key' value from the resulting page.
  4. Save it locally (outside any git repo):

       mkdir -p ~/.config/cs
       touch ~/.config/cs/secrets.env
       chmod 600 ~/.config/cs/secrets.env
       $EDITOR ~/.config/cs/secrets.env
       # Add: STACK_OVERFLOW_API_KEY=<the-key-you-copied>

     OR, for a one-shot test, export it for the current shell only:

       read -rs STACK_OVERFLOW_API_KEY      # silent — not in shell history
       export STACK_OVERFLOW_API_KEY

  5. Use it:

       vor -so "lvm cannot extend volume"

Quota: 10,000 requests/day with a key (300 anonymous). Each query costs 1.
Cache: 24h on disk at ~/.cache/cs/stackoverflow/ — repeat queries are free.
Privacy: the key never appears in error output, the cache file, or logs
         (verified by tests in internal/stackoverflow). It only travels in
         the outbound HTTPS query string to api.stackexchange.com.
Rotate:  delete or edit the app at https://stackapps.com/apps any time.
Clear cache: rm -rf ~/.cache/cs/stackoverflow/

This whole feature is invisible without a configured key. Default vor stays
fully offline.
`)
}

// openAPIDoc returns a minimal OpenAPI 3.0 description of the REST surface.
// It's a hint, not a complete spec — enough for api-browser tools (Postman,
// Bruno, Insomnia, the OpenAPI viewer) to discover the endpoints. We avoid
// vendoring an OpenAPI schema library to preserve the zero-dep invariant.
func openAPIDoc(version string) map[string]any {
	path := func(method, summary string, params []map[string]any, response200 string) map[string]any {
		op := map[string]any{
			"summary": summary,
			"responses": map[string]any{
				"200": map[string]any{"description": response200},
			},
		}
		if len(params) > 0 {
			op["parameters"] = params
		}
		return map[string]any{strings.ToLower(method): op}
	}
	queryParam := func(name, desc string) map[string]any {
		return map[string]any{
			"name":        name,
			"in":          "query",
			"required":    true,
			"description": desc,
			"schema":      map[string]any{"type": "string"},
		}
	}
	pathParam := func(name, desc string) map[string]any {
		return map[string]any{
			"name":        name,
			"in":          "path",
			"required":    true,
			"description": desc,
			"schema":      map[string]any{"type": "string"},
		}
	}
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "vör (vor) cheatsheet REST API",
			"version":     version,
			"description": "Offline-first cheatsheet CLI exposing its embedded corpus over HTTP. The /api/stackoverflow endpoint is an opt-in bonus that requires STACK_OVERFLOW_API_KEY.",
			"license":     map[string]any{"name": "GPL-3.0-or-later", "url": "https://www.gnu.org/licenses/gpl-3.0.html"},
		},
		"servers": []map[string]any{
			{"url": "http://127.0.0.1:9876", "description": "default local bind"},
		},
		"paths": map[string]any{
			"/api/version":               path("GET", "Build identification", nil, "JSON {binary, version, goos, goarch, go_version}"),
			"/api/health":                path("GET", "Liveness probe", nil, "JSON {status, sheets, categories, details}"),
			"/api/openapi":               path("GET", "This document", nil, "OpenAPI 3.0 hint"),
			"/api/topics":                path("GET", "List all topics", nil, "Array of topic summaries"),
			"/api/topics/{name}":         path("GET", "Get a single sheet", []map[string]any{pathParam("name", "topic slug")}, "Sheet content as JSON"),
			"/api/topics/{name}/detail":  path("GET", "Get the deep-dive detail page if one exists", []map[string]any{pathParam("name", "topic slug")}, "Detail page content"),
			"/api/topics/{name}/related": path("GET", "Get cross-referenced related topics", []map[string]any{pathParam("name", "topic slug")}, "Array of related topic summaries"),
			"/api/categories":            path("GET", "List categories", nil, "Array of category names"),
			"/api/search":                path("GET", "Full-text search", []map[string]any{queryParam("q", "search query")}, "Array of search hits"),
			"/api/compare":               path("GET", "Compare two topics", []map[string]any{queryParam("a", "first topic"), queryParam("b", "second topic")}, "Comparison structure"),
			"/api/calc":                  path("POST", "Evaluate calculator expression", nil, "Calculator result"),
			"/api/subnet":                path("POST", "Run subnet calculator", nil, "Subnet calculation result"),
			"/api/verify/{name}":         path("GET", "Verify math in a detail page", []map[string]any{pathParam("name", "topic slug")}, "Verification status"),
			"/api/stats":                 path("GET", "Per-category statistics", nil, "Stats object"),
			"/api/bookmarks":             path("GET", "List bookmarks", nil, "Array of bookmarked topic names"),
			"/api/bookmarks/{name}":      path("POST", "Toggle bookmark on/off", []map[string]any{pathParam("name", "topic slug")}, "Updated bookmark state"),
			"/api/stackoverflow":         path("GET", "Bonus opt-in: live Stack Overflow lookup. Requires STACK_OVERFLOW_API_KEY (env or ~/.config/cs/secrets.env). 503 if unconfigured.", []map[string]any{queryParam("q", "search query")}, "Stack Exchange Result"),
		},
	}
}
