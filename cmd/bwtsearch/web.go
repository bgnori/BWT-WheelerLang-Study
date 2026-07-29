package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
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
	if err := fs.Parse(args); err != nil {
		return err
	}

	idx, err := loadIndex(*indexPath)
	if err != nil {
		return err
	}

	app := webApp{
		idx:            idx,
		defaultLimit:   *defaultLimit,
		defaultContext: *defaultContext,
		indexPath:      *indexPath,
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
	idx            *fmindex.Index
	defaultLimit   int
	defaultContext int
	indexPath      string
}

type apiError struct {
	Error string `json:"error"`
}

type infoResponse struct {
	IndexPath    string `json:"indexPath"`
	TextLength   int    `json:"textLength"`
	SuffixLength int    `json:"suffixLength"`
	AlphabetSize int    `json:"alphabetSize"`
	DefaultLimit int    `json:"defaultLimit"`
	ContextSize  int    `json:"contextSize"`
}

type matchResponse struct {
	Position int    `json:"position"`
	Snippet  string `json:"snippet"`
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
		IndexPath:    a.indexPath,
		TextLength:   a.idx.TextLen(),
		SuffixLength: a.idx.SALen(),
		AlphabetSize: a.idx.AlphabetSize(),
		DefaultLimit: a.defaultLimit,
		ContextSize:  a.defaultContext,
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

	matches := make([]matchResponse, 0, len(positions))
	for _, pos := range positions {
		snippet := normalizeSnippet(a.idx.ContextAround(pos, len(q), contextSize))
		matches = append(matches, matchResponse{Position: pos, Snippet: snippet})
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
      grid-template-columns: 1fr 120px 120px 130px;
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

      document.getElementById("limit").value = String(info.defaultLimit);
      document.getElementById("context").value = String(info.contextSize);
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
        return '<li>' +
          '<span class="pos">pos ' + esc(String(m.position)) + '</span>' +
          '<span class="snippet">' + esc(m.snippet) + '</span>' +
        '</li>';
      }).join("");
    }

    form.addEventListener("submit", async (ev) => {
      ev.preventDefault();
      setStatus("Searching...");
      resultHead.textContent = "";
      results.innerHTML = "";
      empty.style.display = "none";

      const query = document.getElementById("query").value.trim();
      const limit = document.getElementById("limit").value;
      const context = document.getElementById("context").value;
      if (!query) {
        setStatus("Pattern を入力してください。", true);
        return;
      }

      const params = new URLSearchParams({ q: query, limit, context });
  const res = await fetch('/api/search?' + params.toString());
      const body = await res.json();

      if (!res.ok) {
        setStatus(body.error || "Search failed", true);
        empty.style.display = "block";
        empty.textContent = "エラー内容を確認してクエリを修正してください。";
        return;
      }

      setStatus("");
      renderResults(body);
    });

    loadInfo().catch((err) => {
      setStatus('初期化エラー: ' + err.message, true);
    });
  </script>
</body>
</html>
`
