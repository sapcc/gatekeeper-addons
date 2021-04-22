<!DOCTYPE html>
<link rel="preload" href="/static/style.css" as="style">
<header>
  <img src="/static/logo.svg">
  <h1>
    Decentralized Observer Of Policies
    <small><a href="https://futurama.fandom.com/wiki/Democratic_Order_of_Planets">[huh?]</a></small>
  </h1>
</header>
{{ range $cluster, $report := . }}
<h2>{{ $cluster }}</h2>
<pre><code>{{ jsonIndent $report }}</code></pre>
{{ end }}
<link rel="stylesheet" href="/static/style.css" preload>
