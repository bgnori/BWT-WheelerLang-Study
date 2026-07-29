package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"regexp/syntax"
	"sort"
	"strconv"
	"strings"

	"github.com/bgnori/bwt-wheelerlang-study/internal/fmindex"
	"github.com/bgnori/bwt-wheelerlang-study/internal/starfree"
)

func runWeb(args []string) error {
	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	addr := fs.String("addr", ":8080", "listen address")
	indexPath := fs.String("index", "data/moby_dick.idx", "path to FM-index file")
	defaultLimit := fs.Int("limit", 20, "default maximum number of results")
	defaultContext := fs.Int("context", 80, "default context size")
	minChars := fs.Int("min-chars", 4, "minimum query length to trigger interactive search")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *minChars < 1 {
		*minChars = 1
	}

	idx, err := loadIndex(*indexPath)
	if err != nil {
		return err
	}

	app := webApp{
		idx:                 idx,
		defaultLimit:        *defaultLimit,
		defaultContext:      *defaultContext,
		minInteractiveChars: *minChars,
		indexPath:           *indexPath,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.handleHome)
	mux.HandleFunc("/api/info", app.handleInfo)
	mux.HandleFunc("/api/search", app.handleSearch)

	fmt.Printf("web app running on http://localhost%s\n", *addr)
	fmt.Printf("loaded index: %s (text length: %d)\n", *indexPath, idx.TextLen())

	return http.ListenAndServe(*addr, mux)
}

type webApp struct {
	idx                 *fmindex.Index
	defaultLimit        int
	defaultContext      int
	minInteractiveChars int
	indexPath           string
}

type apiError struct {
	Error string `json:"error"`
}

type infoResponse struct {
	IndexPath           string `json:"indexPath"`
	TextLength          int    `json:"textLength"`
	SuffixLength        int    `json:"suffixLength"`
	AlphabetSize        int    `json:"alphabetSize"`
	DefaultLimit        int    `json:"defaultLimit"`
	ContextSize         int    `json:"contextSize"`
	MinInteractiveChars int    `json:"minInteractiveChars"`
}

type matchResponse struct {
	Position int      `json:"position"`
	Snippet  string   `json:"snippet"`
	Before   string   `json:"before"`
	Matched  string   `json:"matched"`
	After    string   `json:"after"`
	Choices  []string `json:"choices"`
}

type searchResponse struct {
	Pattern    string          `json:"pattern"`
	TotalCount int             `json:"totalCount"`
	Truncated  bool            `json:"truncated"`
	Limit      int             `json:"limit"`
	Context    int             `json:"context"`
	Matches    []matchResponse `json:"matches"`
}

func (a webApp) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, webHTML)
}

func (a webApp) handleInfo(w http.ResponseWriter, _ *http.Request) {
	resp := infoResponse{
		IndexPath:           a.indexPath,
		TextLength:          a.idx.TextLen(),
		SuffixLength:        a.idx.SALen(),
		AlphabetSize:        a.idx.AlphabetSize(),
		DefaultLimit:        a.defaultLimit,
		ContextSize:         a.defaultContext,
		MinInteractiveChars: a.minInteractiveChars,
	}
	a.writeJSON(w, http.StatusOK, resp)
}

func (a webApp) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		a.writeJSON(w, http.StatusBadRequest, apiError{Error: "query parameter q is required"})
		return
	}

	limit := parsePositiveInt(r.URL.Query().Get("limit"), a.defaultLimit)
	contextSize := parsePositiveInt(r.URL.Query().Get("context"), a.defaultContext)

	res, err := starfree.Search(a.idx, q, limit)
	if err != nil {
		a.writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	positions := res.Positions(a.idx)
	sort.Ints(positions)

	re, err := syntax.Parse(q, syntax.Perl)
	if err != nil {
		a.writeJSON(w, http.StatusBadRequest, apiError{Error: fmt.Sprintf("invalid regex: %v", err)})
		return
	}
	text := fullTextBytes(a.idx)

	matches := make([]matchResponse, 0, len(positions))
	for _, pos := range positions {
		matchLen, choices := explainMatchAt(re, text, pos)
		before, matched, after := splitContext(text, pos, matchLen, contextSize)
		snippet := before + matched + after
		matches = append(matches, matchResponse{
			Position: pos,
			Snippet:  snippet,
			Before:   before,
			Matched:  matched,
			After:    after,
			Choices:  choices,
		})
	}

	a.writeJSON(w, http.StatusOK, searchResponse{
		Pattern:    q,
		TotalCount: res.TotalCount,
		Truncated:  res.Truncated,
		Limit:      limit,
		Context:    contextSize,
		Matches:    matches,
	})
}

func (a webApp) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

type traceState struct {
	end     int
	choices []string
}

func fullTextBytes(idx *fmindex.Index) []byte {
	// ContextAround can return the full text when the requested range spans it all.
	return []byte(idx.ContextAround(0, idx.TextLen(), 0))
}

func explainMatchAt(re *syntax.Regexp, text []byte, pos int) (int, []string) {
	states := traceRegex(re, text, pos)
	if len(states) == 0 {
		return 0, nil
	}

	best := states[0]
	for _, st := range states[1:] {
		if st.end > best.end {
			best = st
		}
	}

	matchLen := best.end - pos
	if matchLen < 0 {
		matchLen = 0
	}
	return matchLen, best.choices
}

func traceRegex(re *syntax.Regexp, text []byte, pos int) []traceState {
	const maxStates = 256
	switch re.Op {
	case syntax.OpLiteral:
		cur := pos
		for _, r := range re.Rune {
			if cur >= len(text) {
				return nil
			}
			if r > 127 {
				return nil
			}
			b := byte(r)
			ch := text[cur]
			if re.Flags&syntax.FoldCase != 0 {
				if !equalFoldASCII(ch, b) {
					return nil
				}
			} else if ch != b {
				return nil
			}
			cur++
		}
		return []traceState{{end: cur}}

	case syntax.OpCharClass:
		if pos >= len(text) {
			return nil
		}
		ch := text[pos]
		if !matchesCharClass(re.Rune, ch) {
			return nil
		}
		choice := fmt.Sprintf("文字クラス %s -> %q", re.String(), printableByte(ch))
		return []traceState{{end: pos + 1, choices: []string{choice}}}

	case syntax.OpAnyChar:
		if pos >= len(text) {
			return nil
		}
		return []traceState{{end: pos + 1}}

	case syntax.OpAnyCharNotNL:
		if pos >= len(text) || text[pos] == '\n' {
			return nil
		}
		return []traceState{{end: pos + 1}}

	case syntax.OpConcat:
		states := []traceState{{end: pos}}
		for _, sub := range re.Sub {
			next := make([]traceState, 0)
			for _, st := range states {
				subs := traceRegex(sub, text, st.end)
				for _, ss := range subs {
					next = append(next, traceState{
						end:     ss.end,
						choices: appendChoices(st.choices, ss.choices),
					})
					if len(next) >= maxStates {
						break
					}
				}
				if len(next) >= maxStates {
					break
				}
			}
			if len(next) == 0 {
				return nil
			}
			states = next
		}
		return states

	case syntax.OpAlternate:
		out := make([]traceState, 0)
		total := len(re.Sub)
		for i, sub := range re.Sub {
			subs := traceRegex(sub, text, pos)
			branch := fmt.Sprintf("分岐 %d/%d を採用: %s", i+1, total, sub.String())
			for _, ss := range subs {
				out = append(out, traceState{
					end:     ss.end,
					choices: append([]string{branch}, ss.choices...),
				})
				if len(out) >= maxStates {
					return out
				}
			}
		}
		return out

	case syntax.OpQuest:
		out := []traceState{{end: pos, choices: []string{"オプション ? を省略"}}}
		if len(re.Sub) > 0 {
			subs := traceRegex(re.Sub[0], text, pos)
			for _, ss := range subs {
				out = append(out, traceState{
					end:     ss.end,
					choices: append([]string{"オプション ? を採用"}, ss.choices...),
				})
			}
		}
		return out

	case syntax.OpRepeat:
		if len(re.Sub) == 0 {
			return []traceState{{end: pos}}
		}
		max := re.Max
		if max < re.Min {
			max = re.Min
		}
		out := make([]traceState, 0)
		for c := max; c >= re.Min; c-- {
			states := []traceState{{end: pos}}
			ok := true
			for i := 0; i < c; i++ {
				next := make([]traceState, 0)
				for _, st := range states {
					subs := traceRegex(re.Sub[0], text, st.end)
					for _, ss := range subs {
						if ss.end == st.end {
							continue
						}
						next = append(next, traceState{
							end:     ss.end,
							choices: appendChoices(st.choices, ss.choices),
						})
						if len(next) >= maxStates {
							break
						}
					}
					if len(next) >= maxStates {
						break
					}
				}
				if len(next) == 0 {
					ok = false
					break
				}
				states = next
			}
			if !ok {
				continue
			}

			repeatChoice := fmt.Sprintf("反復 %s で %d 回を採用", re.String(), c)
			for _, st := range states {
				st.choices = append([]string{repeatChoice}, st.choices...)
				out = append(out, st)
				if len(out) >= maxStates {
					return out
				}
			}
		}
		return out

	case syntax.OpCapture:
		if len(re.Sub) == 0 {
			return []traceState{{end: pos}}
		}
		return traceRegex(re.Sub[0], text, pos)

	case syntax.OpEmptyMatch,
		syntax.OpBeginText, syntax.OpEndText,
		syntax.OpBeginLine, syntax.OpEndLine,
		syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return []traceState{{end: pos}}
	}

	return nil
}

func appendChoices(a, b []string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	out := make([]string, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	return out
}

func equalFoldASCII(a, b byte) bool {
	if a == b {
		return true
	}
	if a >= 'A' && a <= 'Z' {
		a = a + 32
	}
	if b >= 'A' && b <= 'Z' {
		b = b + 32
	}
	return a == b
}

func matchesCharClass(ranges []rune, ch byte) bool {
	r := rune(ch)
	for i := 0; i+1 < len(ranges); i += 2 {
		if r >= ranges[i] && r <= ranges[i+1] {
			return true
		}
	}
	return false
}

func printableByte(ch byte) string {
	if ch >= 32 && ch <= 126 {
		return string([]byte{ch})
	}
	return fmt.Sprintf("\\x%02x", ch)
}

func splitContext(text []byte, pos int, matchLen int, contextSize int) (string, string, string) {
	n := len(text)
	if pos < 0 {
		pos = 0
	}
	if pos > n {
		pos = n
	}
	if matchLen < 0 {
		matchLen = 0
	}
	if pos+matchLen > n {
		matchLen = n - pos
	}

	start := pos - contextSize
	if start < 0 {
		start = 0
	}
	end := pos + matchLen + contextSize
	if end > n {
		end = n
	}

	before := flattenWhitespace(string(text[start:pos]))
	matched := flattenWhitespace(string(text[pos : pos+matchLen]))
	after := flattenWhitespace(string(text[pos+matchLen : end]))
	if matched == "" {
		matched = "(empty match)"
	}
	return before, matched, after
}

func flattenWhitespace(s string) string {
	rep := strings.NewReplacer("\r", " ", "\n", " ", "\t", " ")
	return rep.Replace(s)
}

const webHTML = `<!doctype html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>BWT Search Playground</title>
  <style>
    :root {
      --bg: #f6f3ea;
      --paper: rgba(255, 255, 255, 0.88);
      --ink: #1a1f16;
      --accent: #0a7c66;
      --accent-soft: #d4efe8;
      --line: #cad5be;
      --warn: #9f3a1c;
    }

    * { box-sizing: border-box; }

    body {
      margin: 0;
      min-height: 100vh;
      color: var(--ink);
      font-family: "IBM Plex Sans", "Noto Sans JP", sans-serif;
      background:
        radial-gradient(circle at 16% 18%, rgba(10, 124, 102, 0.12), transparent 40%),
        radial-gradient(circle at 86% 10%, rgba(181, 99, 26, 0.15), transparent 36%),
        linear-gradient(135deg, #efe7d2 0%, #f6f3ea 65%, #eef4e6 100%);
      padding: 24px;
    }

    .shell {
      max-width: 1040px;
      margin: 0 auto;
      display: grid;
      gap: 18px;
      animation: lift 460ms ease-out both;
    }

    .hero {
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: 20px;
      padding: 24px;
      backdrop-filter: blur(8px);
      box-shadow: 0 16px 36px rgba(42, 59, 31, 0.12);
    }

    h1 {
      margin: 0;
      font-family: "Alegreya", "Noto Serif JP", serif;
      letter-spacing: 0.01em;
      font-size: clamp(1.6rem, 3vw, 2.4rem);
      line-height: 1.2;
    }

    .sub {
      margin-top: 8px;
      font-size: 0.95rem;
      opacity: 0.85;
    }

    .grid {
      display: grid;
      gap: 12px;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      margin-top: 16px;
    }

    .metric {
      background: var(--accent-soft);
      border-radius: 12px;
      border: 1px solid rgba(10, 124, 102, 0.2);
      padding: 12px;
    }

    .metric .k {
      font-size: 0.75rem;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      opacity: 0.75;
    }

    .metric .v {
      margin-top: 4px;
      font-weight: 700;
      font-size: 1.1rem;
    }

    .panel {
      background: rgba(255, 255, 255, 0.9);
      border: 1px solid var(--line);
      border-radius: 20px;
      padding: 18px;
      box-shadow: 0 8px 24px rgba(52, 59, 47, 0.08);
    }

    form {
	display: grid;
	gap: 10px;
	grid-template-columns: 1fr 120px 120px 120px 130px;
      align-items: end;
    }

    label {
      display: block;
      font-size: 0.78rem;
      letter-spacing: 0.03em;
      margin-bottom: 4px;
      opacity: 0.88;
    }

    input {
      width: 100%;
      padding: 11px 12px;
      border-radius: 10px;
      border: 1px solid var(--line);
      background: #fffefa;
      color: var(--ink);
      font: inherit;
    }

    button {
      height: 43px;
      border: 0;
      border-radius: 10px;
      font-weight: 700;
      color: white;
      cursor: pointer;
      background: linear-gradient(135deg, #0a7c66, #155f84);
      transition: transform 120ms ease, filter 120ms ease;
    }

    button:hover {
      filter: brightness(1.06);
      transform: translateY(-1px);
    }

    .status {
      min-height: 1.4em;
      margin-top: 10px;
      font-size: 0.92rem;
    }

    .status.error { color: var(--warn); }

    .result-head {
      margin-top: 16px;
      font-size: 0.88rem;
      opacity: 0.84;
    }

    ol {
      margin: 10px 0 0;
      padding-left: 22px;
      display: grid;
      gap: 8px;
    }

    li {
      background: #fcfcf8;
      border: 1px solid var(--line);
      border-radius: 10px;
      padding: 10px;
      line-height: 1.45;
      animation: fadeIn 260ms ease both;
    }

    .pos {
      font-family: "IBM Plex Mono", monospace;
      color: #155f84;
      font-size: 0.82rem;
      margin-bottom: 4px;
      display: block;
    }

    .snippet {
      word-break: break-word;
      font-family: "Alegreya Sans", "Noto Sans JP", sans-serif;
    }

    mark.hit {
      background: #ffd76b;
      color: #2a1f00;
      border-radius: 4px;
      padding: 0 3px;
      font-weight: 700;
    }

    .trace-title {
      margin-top: 8px;
      font-size: 0.78rem;
      opacity: 0.8;
      letter-spacing: 0.04em;
      text-transform: uppercase;
    }

    ul.trace {
      margin: 6px 0 0;
      padding-left: 18px;
      display: grid;
      gap: 4px;
    }

    ul.trace li {
      background: transparent;
      border: 0;
      border-radius: 0;
      padding: 0;
      line-height: 1.35;
      animation: none;
      font-size: 0.9rem;
      color: #324041;
    }

    .empty {
      margin-top: 12px;
      opacity: 0.8;
    }

    @keyframes lift {
      from { opacity: 0; transform: translateY(14px); }
      to { opacity: 1; transform: translateY(0); }
    }

    @keyframes fadeIn {
      from { opacity: 0; transform: translateX(-8px); }
      to { opacity: 1; transform: translateX(0); }
    }

    @media (max-width: 860px) {
      body { padding: 14px; }
      form { grid-template-columns: 1fr 1fr; }
      .grid { grid-template-columns: 1fr; }
      button { grid-column: span 2; }
    }
  </style>
</head>
<body>
  <main class="shell">
    <section class="hero">
      <h1>FM-index / Star-free Search Playground</h1>
      <div class="sub">星なし正規表現で、インデックス検索の挙動をブラウザで試せます。</div>
      <div class="grid" id="metrics"></div>
    </section>

    <section class="panel">
      <form id="searchForm">
        <div>
          <label for="query">Pattern</label>
          <input id="query" name="query" placeholder="例: whale|ship" required>
        </div>
        <div>
          <label for="limit">Limit</label>
          <input id="limit" name="limit" type="number" min="1" value="20">
        </div>
        <div>
          <label for="context">Context</label>
          <input id="context" name="context" type="number" min="1" value="80">
        </div>
				<div>
					<label for="minChars">Auto From</label>
					<input id="minChars" name="minChars" type="number" min="1" value="4">
				</div>
        <button type="submit">Search</button>
      </form>
      <div id="status" class="status"></div>
      <div id="resultHead" class="result-head"></div>
      <ol id="results"></ol>
      <div id="empty" class="empty">検索語を入力して実行してください。</div>
    </section>
  </main>

  <script>
    const metrics = document.getElementById("metrics");
    const form = document.getElementById("searchForm");
    const statusEl = document.getElementById("status");
    const resultHead = document.getElementById("resultHead");
    const results = document.getElementById("results");
    const empty = document.getElementById("empty");
		const queryInput = document.getElementById("query");
		const limitInput = document.getElementById("limit");
		const contextInput = document.getElementById("context");
		const minCharsInput = document.getElementById("minChars");

		const state = {
			minInteractiveChars: 4,
			debounceMs: 220,
			requestSeq: 0,
		};
		let debounceTimer = 0;

    function esc(s) {
      return s
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;")
        .replaceAll('"', "&quot;")
        .replaceAll("'", "&#039;");
    }

    function setStatus(msg, isError = false) {
      statusEl.textContent = msg || "";
      statusEl.className = isError ? "status error" : "status";
    }

    async function loadInfo() {
      const res = await fetch("/api/info");
      const info = await res.json();
      metrics.innerHTML = [
        ["Text length", info.textLength],
        ["Suffix array", info.suffixLength],
        ["Alphabet", info.alphabetSize],
      ].map(([k, v]) => {
        return '<div class="metric">' +
          '<div class="k">' + esc(String(k)) + '</div>' +
          '<div class="v">' + esc(String(v)) + '</div>' +
        '</div>';
      }).join("");

			limitInput.value = String(info.defaultLimit);
			contextInput.value = String(info.contextSize);
			state.minInteractiveChars = Math.max(1, Number(info.minInteractiveChars || 4));
			minCharsInput.value = String(state.minInteractiveChars);

			setStatus("" + state.minInteractiveChars + "文字以上で自動検索します。");
    }

		function clearResults(message) {
			resultHead.textContent = "";
			results.innerHTML = "";
			empty.style.display = "block";
			empty.textContent = message;
		}

		function getMinChars() {
			const n = Number(minCharsInput.value);
			if (!Number.isFinite(n) || n < 1) {
				return state.minInteractiveChars;
			}
			return Math.floor(n);
		}

		async function executeSearch(manual) {
			const query = queryInput.value.trim();
			const limit = limitInput.value;
			const context = contextInput.value;
			const minChars = getMinChars();
			state.minInteractiveChars = minChars;

			if (!query) {
				clearResults("検索語を入力して実行してください。");
				setStatus("", false);
				return;
			}
			if (!manual && query.length < minChars) {
				clearResults(minChars + "文字以上で自動検索を開始します。現在: " + query.length + "文字");
				setStatus("入力待ち", false);
				return;
			}

			const reqID = ++state.requestSeq;
			setStatus("Searching...");
			const params = new URLSearchParams({ q: query, limit, context });
			const res = await fetch('/api/search?' + params.toString());
			const body = await res.json();

			if (reqID !== state.requestSeq) {
				return;
			}

			if (!res.ok) {
				setStatus(body.error || "Search failed", true);
				clearResults("エラー内容を確認してクエリを修正してください。");
				return;
			}

			setStatus("");
			renderResults(body);
		}

		function queueInteractiveSearch() {
			window.clearTimeout(debounceTimer);
			debounceTimer = window.setTimeout(() => {
				executeSearch(false).catch((err) => {
					setStatus('検索エラー: ' + err.message, true);
				});
			}, state.debounceMs);
		}

    function renderResults(payload) {
      resultHead.textContent = 'pattern="' + payload.pattern + '" / ' + payload.totalCount + ' hit(s)' +
        (payload.truncated ? ' (showing first ' + payload.limit + ')' : '');

      if (!payload.matches || payload.matches.length === 0) {
        results.innerHTML = "";
        empty.textContent = "一致する結果はありません。";
        empty.style.display = "block";
        return;
      }

      empty.style.display = "none";
      results.innerHTML = payload.matches.map(m => {
        const matched = m.matched || "";
        const hit = '<mark class="hit">' + esc(matched) + '</mark>';
        const traceItems = (m.choices || []).map(c => '<li>' + esc(c) + '</li>').join('');
        const traceBlock = traceItems
          ? '<div class="trace-title">採用された選択肢</div><ul class="trace">' + traceItems + '</ul>'
          : '';

        return '<li>' +
          '<span class="pos">pos ' + esc(String(m.position)) + '</span>' +
          '<span class="snippet">' + esc(m.before || '') + hit + esc(m.after || '') + '</span>' +
          traceBlock +
        '</li>';
      }).join("");
    }

    form.addEventListener("submit", async (ev) => {
			ev.preventDefault();
			executeSearch(true).catch((err) => {
				setStatus('検索エラー: ' + err.message, true);
			});
    });

		queryInput.addEventListener("input", queueInteractiveSearch);

		limitInput.addEventListener("input", () => {
			if (queryInput.value.trim().length >= getMinChars()) {
				queueInteractiveSearch()
			}
		});

		contextInput.addEventListener("input", () => {
			if (queryInput.value.trim().length >= getMinChars()) {
				queueInteractiveSearch()
			}
		});

		minCharsInput.addEventListener("input", () => {
			const minChars = getMinChars();
			state.minInteractiveChars = minChars;
			if (queryInput.value.trim().length >= minChars) {
				queueInteractiveSearch()
			} else {
				clearResults(minChars + "文字以上で自動検索を開始します。現在: " + queryInput.value.trim().length + "文字");
			}
		});

    loadInfo().catch((err) => {
      setStatus('初期化エラー: ' + err.message, true);
    });
  </script>
</body>
</html>
`
