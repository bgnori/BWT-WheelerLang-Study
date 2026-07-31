package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	bwtsearch "github.com/bgnori/bwt-wheelerlang-study"
	"golang.org/x/term"
)

// anyIndex is satisfied by both *bwtsearch.Index (FM-index) and
// *bwtsearch.StdlibIndex (stdlib suffix array).
type anyIndex interface {
	TextLen() int
	ContextAround(pos, patLen, ctxSize int) string
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "build":
		err = runBuild(os.Args[2:])
	case "build-multi":
		err = runBuildMulti(os.Args[2:])
	case "info":
		err = runInfo(os.Args[2:])
	case "graph":
		err = runGraph(os.Args[2:])
	case "browse":
		err = runBrowse(os.Args[2:])
	case "search":
		err = runSearch(os.Args[2:])
	case "web":
		err = runWeb(os.Args[2:])
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "bwtsearch <command> [args]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  build [--algo doubling|sais|suffixarray] [--occ bitvectors|wavelet] <input-file> <index-file>")
	fmt.Fprintln(os.Stderr, "  build-multi [--algo doubling|sais|suffixarray] [--occ bitvectors|wavelet] <index-file> <file1> [file2 ...]")
	fmt.Fprintln(os.Stderr, "  info <index-file>")
	fmt.Fprintln(os.Stderr, "  graph [flags] <index-file>")
	fmt.Fprintln(os.Stderr, "  browse <index-file> [--show N] [--context N]")
	fmt.Fprintln(os.Stderr, "  search [flags] <index-file> <pattern>")
	fmt.Fprintln(os.Stderr, "  web [--index FILE] [--addr ADDR] [--limit N] [--context N] [--min-chars N]")
}

func runBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	algoFlag := fs.String("algo", "doubling", "suffix-array construction algorithm: doubling, sais, or suffixarray")
	occFlag := fs.String("occ", "bitvectors", "occurrence-array structure: bitvectors or wavelet")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("usage: bwtsearch build [--algo doubling|sais|suffixarray] [--occ bitvectors|wavelet] <input-file> <index-file>")
	}

	inputPath := fs.Arg(0)
	indexPath := fs.Arg(1)
	text, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	if strings.ToLower(*algoFlag) == "suffixarray" {
		idx := bwtsearch.BuildStdlib(text)
		if err := idx.Save(indexPath); err != nil {
			return err
		}
		fmt.Printf("built stdlib suffix array index: %s (%d bytes)\n", indexPath, idx.TextLen())
		return nil
	}

	algo, err := parseAlgo(*algoFlag)
	if err != nil {
		return err
	}
	occ, err := parseOcc(*occFlag)
	if err != nil {
		return err
	}

	idx := bwtsearch.BuildWithOptions(text, algo, occ)
	out, err := os.Create(indexPath)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	defer out.Close()

	if _, err := idx.WriteTo(out); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	fmt.Printf("built index: %s (%d bytes)\n", indexPath, idx.TextLen())
	return nil
}

func runBuildMulti(args []string) error {
	fs := flag.NewFlagSet("build-multi", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	algoFlag := fs.String("algo", "doubling", "suffix-array construction algorithm: doubling, sais, or suffixarray")
	occFlag := fs.String("occ", "bitvectors", "occurrence-array structure: bitvectors or wavelet")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: bwtsearch build-multi [--algo doubling|sais|suffixarray] [--occ bitvectors|wavelet] <index-file> <file1> [file2 ...]")
	}

	indexPath := fs.Arg(0)
	var texts [][]byte
	for _, path := range fs.Args()[1:] {
		text, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read input %s: %w", path, err)
		}
		texts = append(texts, text)
	}

	if strings.ToLower(*algoFlag) == "suffixarray" {
		idx := bwtsearch.BuildStdlibFromFiles(texts, nil)
		if err := idx.Save(indexPath); err != nil {
			return err
		}
		fmt.Printf("built stdlib suffix array index from %d files: %s (%d bytes)\n", len(texts), indexPath, idx.TextLen())
		return nil
	}

	algo, err := parseAlgo(*algoFlag)
	if err != nil {
		return err
	}
	occ, err := parseOcc(*occFlag)
	if err != nil {
		return err
	}

	idx := bwtsearch.BuildFromFilesWithOptions(texts, nil, algo, occ)
	out, err := os.Create(indexPath)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	defer out.Close()

	if _, err := idx.WriteTo(out); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	fmt.Printf("built index from %d files: %s (%d bytes)\n", len(texts), indexPath, idx.TextLen())
	return nil
}

func runInfo(args []string) error {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: bwtsearch info <index-file>")
	}

	idx, err := loadAnyIndex(fs.Arg(0))
	if err != nil {
		return err
	}

	switch fi := idx.(type) {
	case *bwtsearch.Index:
		fmt.Printf("backend: fm-index\n")
		fmt.Printf("text length: %d\n", fi.TextLen())
		fmt.Printf("sa length: %d\n", fi.SALen())
		fmt.Printf("alphabet size: %d\n", fi.AlphabetSize())
	case *bwtsearch.StdlibIndex:
		fmt.Printf("backend: stdlib suffixarray\n")
		fmt.Printf("text length: %d\n", fi.TextLen())
	}
	return nil
}

func runGraph(args []string) error {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	maxNodes := fs.Int("max-nodes", 16, "maximum number of nodes to render (0 = all)")
	markdown := fs.Bool("markdown", true, "wrap output in a mermaid code block")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: bwtsearch graph [flags] <index-file>")
	}

	anyIdx, err := loadAnyIndex(fs.Arg(0))
	if err != nil {
		return err
	}
	idx, ok := anyIdx.(*bwtsearch.Index)
	if !ok {
		return fmt.Errorf("graph command requires an FM-index (index was built with --algo suffixarray)")
	}

	graph := idx.WheelerGraphMermaid(*maxNodes)
	if *markdown {
		fmt.Println("```mermaid")
		fmt.Print(graph)
		fmt.Println("```")
		return nil
	}

	fmt.Print(graph)
	return nil
}

func runBrowse(args []string) error {
	fs := flag.NewFlagSet("browse", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	show := fs.Int("show", 10, "number of matches to show")
	context := fs.Int("context", 40, "context size")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: bwtsearch browse <index-file> [--show N] [--context N]")
	}

	idx, err := loadAnyIndex(fs.Arg(0))
	if err != nil {
		return err
	}

	if term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
		return runBrowseInteractive(idx, *show, *context, os.Stdin, os.Stdout)
	}

	return runBrowseLineMode(idx, *show, *context, os.Stdin, os.Stdout)
}

func runBrowseLineMode(idx anyIndex, show int, context int, in io.Reader, out io.Writer) error {
	reader := bufio.NewScanner(in)
	for {
		fmt.Fprint(out, "> ")
		if !reader.Scan() {
			return reader.Err()
		}

		pattern := strings.TrimSpace(reader.Text())
		if pattern == "" {
			return nil
		}

		if err := printBrowseMatches(out, idx, pattern, show, context); err != nil {
			return err
		}
	}
}

func runBrowseInteractive(idx anyIndex, show int, context int, in *os.File, out io.Writer) error {
	m := newBrowseModel(idx, show, context)
	p := tea.NewProgram(m, tea.WithInput(in), tea.WithOutput(out))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("interactive browse: %w", err)
	}
	return nil
}

func printBrowseMatches(out io.Writer, idx anyIndex, pattern string, show int, context int) error {
	positions, truncated, err := searchAny(idx, pattern, show)
	if err != nil {
		_, writeErr := fmt.Fprintln(out, err)
		if writeErr != nil {
			return writeErr
		}
		return nil
	}

	sort.Ints(positions)
	for _, pos := range positions {
		snippet := normalizeSnippet(idx.ContextAround(pos, len(pattern), context))
		fmt.Fprintf(out, "%d: %s\n", pos, snippet)
	}
	if len(positions) == 0 {
		fmt.Fprintln(out, "(no matches)")
	}
	if truncated {
		fmt.Fprintf(out, "(truncated to %d results)\n", show)
	}

	return nil
}

func normalizeSnippet(s string) string {
	// Collapse newlines/tabs/repeated spaces so each match always renders on one line.
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

type browseModel struct {
	idx       anyIndex
	show      int
	context   int
	input     textinput.Model
	results   []string
	truncated bool
	errText   string
}

func newBrowseModel(idx anyIndex, show int, context int) browseModel {
	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.Prompt = "> "
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 48

	return browseModel{
		idx:     idx,
		show:    show,
		context: context,
		input:   ti,
	}
}

func (m browseModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m browseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d", "esc":
			return m, tea.Quit
		case "enter":
			if strings.TrimSpace(m.input.Value()) == "" {
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.refreshResults()
	return m, cmd
}

func (m *browseModel) refreshResults() {
	pattern := strings.TrimSpace(m.input.Value())
	m.results = m.results[:0]
	m.truncated = false
	m.errText = ""
	if pattern == "" {
		return
	}

	positions, truncated, err := searchAny(m.idx, pattern, m.show)
	if err != nil {
		m.errText = err.Error()
		return
	}

	sort.Ints(positions)
	for _, pos := range positions {
		snippet := normalizeSnippet(m.idx.ContextAround(pos, len(pattern), m.context))
		m.results = append(m.results, fmt.Sprintf("%d: %s", pos, snippet))
	}
	m.truncated = truncated
}

func (m browseModel) View() string {
	var b strings.Builder
	b.WriteString("Interactive browse: type to search, Enter on empty, Esc/Ctrl+C/Ctrl+D to quit\n")
	b.WriteString(m.input.View())
	b.WriteByte('\n')

	if strings.TrimSpace(m.input.Value()) == "" {
		b.WriteString("(type to search)\n")
		return b.String()
	}

	if m.errText != "" {
		b.WriteString(m.errText)
		b.WriteByte('\n')
		return b.String()
	}

	if len(m.results) == 0 {
		b.WriteString("(no matches)\n")
	} else {
		for _, line := range m.results {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	if m.truncated {
		b.WriteString(fmt.Sprintf("(truncated to %d results)\n", m.show))
	}

	return b.String()
}

func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	limit := fs.Int("limit", 20, "maximum number of results")
	context := fs.Int("context", 80, "context size")
	positionsOnly := fs.Bool("positions", false, "print positions only")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("usage: bwtsearch search [flags] <index-file> <pattern>")
	}

	idx, err := loadAnyIndex(fs.Arg(0))
	if err != nil {
		return err
	}
	pattern := fs.Arg(1)

	positions, truncated, err := searchAny(idx, pattern, *limit)
	if err != nil {
		return err
	}

	sort.Ints(positions)
	if *positionsOnly {
		for _, pos := range positions {
			fmt.Println(pos)
		}
	} else {
		for _, pos := range positions {
			fmt.Printf("%d: %s\n", pos, idx.ContextAround(pos, len(pattern), *context))
		}
	}
	if truncated {
		fmt.Fprintf(os.Stderr, "warning: truncated to %d results\n", *limit)
	}
	return nil
}

// searchAny performs a pattern search against idx.
// For FM-index backends, the pattern is interpreted as a star-free regular expression.
// For stdlib suffix array backends, the pattern is matched literally.
func searchAny(idx anyIndex, pattern string, limit int) (positions []int, truncated bool, err error) {
	switch fi := idx.(type) {
	case *bwtsearch.Index:
		res, serr := bwtsearch.Search(fi, pattern, limit)
		if serr != nil {
			return nil, false, serr
		}
		return res.Positions(fi), res.Truncated, nil
	case *bwtsearch.StdlibIndex:
		pos := fi.Locate([]byte(pattern), limit)
		trunc := limit > 0 && len(pos) == limit
		return pos, trunc, nil
	default:
		return nil, false, fmt.Errorf("unsupported index type")
	}
}

func parseAlgo(s string) (bwtsearch.SuffixArrayAlgorithm, error) {
	switch strings.ToLower(s) {
	case "doubling", "":
		return bwtsearch.AlgorithmDoubling, nil
	case "sais":
		return bwtsearch.AlgorithmSAIS, nil
	default:
		return 0, fmt.Errorf("unknown algorithm %q: choose doubling, sais, or suffixarray", s)
	}
}

func parseOcc(s string) (bwtsearch.OccStructure, error) {
	switch strings.ToLower(s) {
	case "bitvectors", "bitvector", "":
		return bwtsearch.OccBitvectors, nil
	case "wavelet", "wavelettree":
		return bwtsearch.OccWaveletTree, nil
	default:
		return 0, fmt.Errorf("unknown occ structure %q: choose bitvectors or wavelet", s)
	}
}

// loadAnyIndex opens path and returns the appropriate index type based on the
// on-disk magic: FMIDX01/FMIDX02 for FM-index, SAIDX01 for stdlib suffix array.
func loadAnyIndex(path string) (anyIndex, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open index: %w", err)
	}
	defer f.Close()

	magic := make([]byte, 7)
	if _, err := io.ReadFull(f, magic); err != nil {
		return nil, fmt.Errorf("read index magic: %w", err)
	}

	r := io.MultiReader(bytes.NewReader(magic), f)
	if string(magic) == "SAIDX01" {
		return bwtsearch.ReadStdlibFrom(r)
	}
	return bwtsearch.ReadFrom(r)
}
