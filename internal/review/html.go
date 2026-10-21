package review

import (
	"html/template"
	"strings"
)

// HTMLPageData is the template data for the self-contained review page.
type HTMLPageData struct {
	Title     string
	Meta      string // one-line provenance summary
	Types     []string
	Sections  map[string]string // resource_type → rendered summary section HTML-ish markdown
	Summary   string
	Checklist string
	Exports   map[string]string // resource_type → HCL ("" in dry-run)
	Snapshots map[string]string // resource_type → JSON ("" in dry-run)
}

// htmlTemplate renders the entire review page. Everything is escaped by
// html/template except code blocks, which pass through pre-escaped helpers.
var htmlTemplate = template.Must(template.New("review").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{.Title}}</title>
<style>
  :root { color-scheme: light dark; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
         margin: 0 auto; max-width: 1200px; padding: 1rem 2rem 4rem; line-height: 1.5; }
  h1 { border-bottom: 2px solid #8884; padding-bottom: .3rem; }
  h2 { border-bottom: 1px solid #8884; padding-bottom: .2rem; margin-top: 2.5rem; }
  table { border-collapse: collapse; margin: .5rem 0 1rem; font-size: .92em; }
  th, td { border: 1px solid #8884; padding: .25rem .6rem; text-align: left; vertical-align: top; }
  th { background: #8881; }
  code, pre { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: .88em; }
  pre { background: #8882; padding: .8rem 1rem; border-radius: 6px; overflow-x: auto; }
  details { margin: .6rem 0; }
  details > summary { cursor: pointer; font-weight: 600; padding: .3rem 0; }
  .meta { color: #888; font-size: .9em; }
  .nav { font-size: .9em; }
  .nav a { margin-right: 1rem; }
  @media (prefers-color-scheme: dark) {
    body { background: #1b1b1f; color: #e4e4e7; }
    th { background: #ffffff14; }
    pre { background: #ffffff12; }
  }
</style>
</head>
<body>
<h1>{{.Title}}</h1>
<p class="meta">{{.Meta}}</p>

<nav class="nav">
  {{range .Types}}<a href="#type-{{.}}">{{.}}</a>{{end}}
  <a href="#checklist">Checklist</a>
  <a href="#summary-md">Summary (markdown)</a>
</nav>

{{range .Types}}
<h2 id="type-{{.}}">{{.}}</h2>

{{with index $.Exports .}}
<details open>
  <summary>Exported HCL</summary>
  <pre>{{.}}</pre>
</details>
{{end}}

{{with index $.Snapshots .}}
<details>
  <summary>Raw API snapshot</summary>
  <pre>{{.}}</pre>
</details>
{{end}}

{{end}}

<h2 id="checklist">Review checklist</h2>
<pre>{{.Checklist}}</pre>

<h2 id="summary-md">Full summary (markdown source)</h2>
<pre>{{.Summary}}</pre>

</body>
</html>
`))

// RenderHTML renders the self-contained review page. Code content (HCL,
// JSON, markdown) is HTML-escaped by html/template automatically; the page
// uses only <pre> wrappers around escaped content.
func RenderHTML(data HTMLPageData) (string, error) {
	var sb strings.Builder
	if err := htmlTemplate.Execute(&sb, data); err != nil {
		return "", err
	}
	return sb.String(), nil
}
