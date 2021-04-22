<!DOCTYPE html>
<link rel="stylesheet" href="/static/style.css">
{{ range $cluster, $report := . }}
<h2>{{ $cluster }}</h2>
<pre><code>{{ printf "%s" $report }}</code></pre>
{{ end }}
