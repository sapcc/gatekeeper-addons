<!DOCTYPE html>
<link rel="preload" href="/static/style.css" as="style">
<h1>
  Decentralized Observer Of Policies
  <small><a href="https://futurama.fandom.com/wiki/Democratic_Order_of_Planets">[huh?]</a></small>
</h1>
{{ range $cluster, $report := . }}
<h2>{{ $cluster }}</h2>
<pre><code>{{ jsonIndent $report }}</code></pre>
{{ end }}
<link rel="stylesheet" href="/static/style.css" preload>
