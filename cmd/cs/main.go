package main

import (
	"flag"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	cs "github.com/bellistech/cs"
	"github.com/bellistech/cs/internal/calc"
	"github.com/bellistech/cs/internal/custom"
	"github.com/bellistech/cs/internal/registry"
	"github.com/bellistech/cs/internal/render"
	"github.com/bellistech/cs/internal/subnet"
)

var version = "dev"

func main() {
	search := flag.String("s", "", "search across all cheatsheets")
	detail := flag.String("d", "", "show deep theory/math for topic")
	list := flag.Bool("l", false, "list all topics with descriptions")
	add := flag.String("add", "", "add custom cheatsheet from file")
	edit := flag.String("edit", "", "open topic in $EDITOR for customization")
	ver := flag.Bool("v", false, "print version")
	random := flag.Bool("random", false, "show a random cheatsheet")
	count := flag.Bool("count", false, "show sheet/category statistics")
	completions := flag.String("completions", "", "generate shell completions (bash, zsh, fish)")
	completionsList := flag.Bool("completions-list", false, "list topics for shell completion (hidden)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `cs - cheatsheet CLI (v%s)

Usage:
  cs                     list all topics grouped by category
  cs <topic>             show cheatsheet (e.g., cs lvm)
  cs <category>          list topics in category (e.g., cs storage)
  cs <topic> <section>   show matching section (e.g., cs lvm extend)
  cs -d <topic>          deep theory/math for topic (e.g., cs -d bgp)
  cs -s <query>          search across all cheatsheets
  cs -l                  list all topics with descriptions
  cs --add <file>        add a custom cheatsheet
  cs --edit <topic>      edit/create custom cheatsheet in $EDITOR

Tools:
  cs calc <expression>   calculator (supports +,-,*,/,%%,**,hex,oct,bin)
  cs subnet <cidr>       subnet calculator (e.g., cs subnet 10.0.0.0/24)

Extra:
  cs --random            show a random cheatsheet
  cs --count             show sheet/category statistics
  cs --completions bash  generate shell completions (bash, zsh, fish)

Options:
`, version)
		flag.PrintDefaults()
	}

	flag.Parse()

	if *ver {
		fmt.Printf("cs %s\n", version)
		os.Exit(0)
	}

	if *completions != "" {
		doCompletions(*completions)
		return
	}

	// Build registry from embedded + custom sheets + detail sheets
	sheetSources := []fs.FS{}
	embedded, err := fs.Sub(cs.EmbeddedSheets, "sheets")
	if err != nil {
		die("embedded sheets: %v", err)
	}
	sheetSources = append(sheetSources, embedded)

	if customFS := custom.Load(); customFS != nil {
		sheetSources = append(sheetSources, customFS)
	}

	detailSources := []fs.FS{}
	detailFS, err := fs.Sub(cs.EmbeddedDetails, "detail")
	if err != nil {
		die("embedded details: %v", err)
	}
	detailSources = append(detailSources, detailFS)

	reg, err := registry.NewWithDetails(sheetSources, detailSources)
	if err != nil {
		die("load sheets: %v", err)
	}

	if *add != "" {
		if err := custom.Add(*add); err != nil {
			die("%v", err)
		}
		return
	}

	if *edit != "" {
		if err := custom.Edit(*edit, cs.EmbeddedSheets); err != nil {
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

	if *detail != "" {
		doDetail(reg, *detail)
		return
	}

	if *search != "" {
		doSearch(reg, *search)
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
		}
	}

	switch len(args) {
	case 0:
		doCategories(reg)
	case 1:
		doShow(reg, args[0])
	default:
		doSection(reg, args[0], strings.Join(args[1:], " "))
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
		sb.WriteString(fmt.Sprintf("  %-*s %-*s %s\n", nameW, s.Name, catW, cat, desc))
	}
	render.PlainOutput(sb.String())
}

func doShow(reg *registry.Registry, name string) {
	// Check for exact sheet match first (takes priority over category)
	s := reg.Get(name)
	if s != nil {
		render.Output(s.Content)
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
	render.Output(s.Content)
	if reg.HasDetail(s.Name) {
		fmt.Fprintf(os.Stderr, "\n\033[1;33mDeep dive available: cs -d %s\033[0m\n", s.Name)
	}
}

func doDetail(reg *registry.Registry, name string) {
	d := reg.GetDetail(name)
	if d != nil {
		render.Output(d.Content)
		return
	}

	// Try fuzzy matching on detail names
	lower := strings.ToLower(name)
	for _, s := range reg.List() {
		if reg.HasDetail(s.Name) && strings.HasPrefix(s.Name, lower) {
			render.Output(reg.GetDetail(s.Name).Content)
			return
		}
	}
	for _, s := range reg.List() {
		if reg.HasDetail(s.Name) && strings.Contains(s.Name, lower) {
			render.Output(reg.GetDetail(s.Name).Content)
			return
		}
	}

	die("no detail available for: %s (use 'cs -l' to list all topics)", name)
}

func doSection(reg *registry.Registry, name, section string) {
	content, err := reg.FindSection(name, section)
	if err != nil {
		die("%v", err)
	}
	render.Output(content)
}

func doSearch(reg *registry.Registry, query string) {
	matches := reg.Search(query)
	if len(matches) == 0 {
		die("no results for: %s", query)
	}

	// Deduplicate by sheet+section
	type key struct{ sheet, section string }
	seen := make(map[key]bool)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Search: %s\n\n", query))

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

Evaluate arithmetic expressions with base conversion.

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

## Functions

    sqrt(x)     square root       cs calc "sqrt(144)"     → 12
    abs(x)      absolute value    cs calc "abs(-42)"      → 42
    log(x)      log base 10      cs calc "log(1000)"     → 3
    ln(x)       natural log       cs calc "ln(e)"         → 1

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
	fmt.Fprintf(os.Stderr, "\033[1;33m🎲 Random: %s [%s]\033[0m\n\n", s.Name, s.Category)
	render.Output(s.Content)
}

func doCount(reg *registry.Registry) {
	sheets := reg.List()
	cats := reg.Categories()

	fmt.Printf("\n\033[1;32mcs statistics\033[0m\n\n")
	fmt.Printf("  %-20s %d\n", "Total sheets:", len(sheets))
	fmt.Printf("  %-20s %d/%d\n", "Detail pages:", reg.DetailCount(), len(sheets))
	fmt.Printf("  %-20s %d\n", "Categories:", len(cats))

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
	switch shell {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	case "fish":
		fmt.Print(fishCompletion)
	default:
		die("unknown shell: %s (supported: bash, zsh, fish)", shell)
	}
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
# Install: cs --completions bash > /usr/local/etc/bash_completion.d/cs
#      or: cs --completions bash >> ~/.bashrc
_cs() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"

    case "$prev" in
        -s|--add|--edit)
            return 0
            ;;
        --completions)
            COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur"))
            return 0
            ;;
    esac

    if [[ "$cur" == -* ]]; then
        COMPREPLY=($(compgen -W "-s -l -v -h --add --edit --random --count --completions --help" -- "$cur"))
        return 0
    fi

    local topics
    topics=$(cs --completions-list 2>/dev/null)
    COMPREPLY=($(compgen -W "$topics" -- "$cur"))
}
complete -F _cs cs
`

const zshCompletion = `#compdef cs
# cs zsh completion
# Install: cs --completions zsh > ~/.zfunc/_cs  (ensure fpath includes ~/.zfunc)
#      or: cs --completions zsh > "${fpath[1]}/_cs"

_cs() {
    local -a topics flags subcommands

    flags=(
        '-s[search across all cheatsheets]:query:'
        '-l[list all topics with descriptions]'
        '-v[print version]'
        '-h[show help]'
        '--add[add custom cheatsheet from file]:file:_files'
        '--edit[edit/create custom cheatsheet in EDITOR]:topic:'
        '--random[show a random cheatsheet]'
        '--count[show sheet/category statistics]'
        '--completions[generate shell completions]:shell:(bash zsh fish)'
    )

    if (( CURRENT == 2 )); then
        topics=("${(@f)$(cs --completions-list 2>/dev/null)}")
        _describe 'topics' topics -- || _arguments $flags
    elif (( CURRENT == 3 )); then
        # Second arg is a section within a topic
        return 0
    fi
}

_cs "$@"
`

const fishCompletion = `# cs fish completion
# Install: cs --completions fish > ~/.config/fish/completions/cs.fish

# Disable file completions
complete -c cs -f

# Flags
complete -c cs -s s -l search -d "Search across all cheatsheets" -r
complete -c cs -s l -d "List all topics with descriptions"
complete -c cs -s v -d "Print version"
complete -c cs -s h -d "Show help"
complete -c cs -l add -d "Add custom cheatsheet from file" -r -F
complete -c cs -l edit -d "Edit/create custom cheatsheet" -r
complete -c cs -l random -d "Show a random cheatsheet"
complete -c cs -l count -d "Show statistics"
complete -c cs -l completions -d "Generate shell completions" -r -a "bash zsh fish"

# Topics and categories (dynamic)
complete -c cs -n "not __fish_seen_subcommand_from (cs --completions-list 2>/dev/null)" \
    -a "(cs --completions-list 2>/dev/null)"
`

func doCalc(expr string) {
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

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "cs: "+format+"\n", args...)
	os.Exit(1)
}
