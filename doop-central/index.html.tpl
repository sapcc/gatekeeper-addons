<!DOCTYPE html>
<title>DOOP: Decentralized Observer of Policies</title>
<link rel="stylesheet" href="/static/style.css">

<header>
  <h1>Decentralized Observer of Policies</h1>
  <div class="buttons">
    {{- range $layer := $.AllClusterLayers }}
      <button type="button" class="selected" data-value="{{ $layer }}">{{ $layer | titlecase }}</button>
    {{- end }}
  </div>
  <div class="buttons">
    {{- range $type := $.AllClusterTypes }}
      <button type="button" class="selected" data-value="{{ $type }}">{{ $type | titlecase }}</button>
    {{- end }}
  </div>
  <input id="search" autofocus type="text" placeholder="Search">
</header>

<main>

  {{- if .Docstrings.Header }}
    <blockquote>{{ .Docstrings.Header }}</blockquote>
  {{- end }}
  <p>
    {{- if .ShowAll }}
    This view includes all constraint configurations. <a href="/">Click here</a> to return to the default view.
    {{- else }}
    This view only includes constraint configurations with the <code>on-prod-ui:&quot;true&quot;</code> label. <a href="/all">Click here</a> for the development view.
    {{- end }}
  </p>

  <section class="folded">
    <h2>Gatekeeper stats (Audit age)</h2>

    <div class="stats">
      {{- range $ctype := $.AllClusterTypes }}
        {{- range $cluster := $.AllClusters }}
          {{- $info := index $.ClusterInfos $cluster }}
          {{- if eq $info.Type $ctype }}
            <table>
              <thead><tr><th class="nobr">{{ $cluster }}</th></tr></thead>
              <tbody><tr><td class="nobr center value-{{ $info.AuditStatus }}">{{ printf "%.1fs" $info.AuditAgeSecs }}</th></tr></tbody>
            </table>
          {{- end }}
        {{- end }}
      {{- end }}
    </div>
  </section>

  {{- range $kind := $.AllTemplateKinds }}
    <section class="check">
      <h2>Check: {{ $kind }}</h2>

      {{- if index $.Docstrings $kind }}
        <blockquote>{{ index $.Docstrings $kind }}</blockquote>
      {{- end }}

      <ul class="violations">
        {{- range $vgroup := index $.ViolationGroups $kind }}
          <li>
            <div class="violation-details">
              {{ $vgroup.Kind }}
              <strong>
                {{- if gt (len $vgroup.Instances) 1 -}}
                  {{- $vgroup.NamePattern | markupPlaceholders -}}
                {{- else -}}
                  {{- $instance := index $vgroup.Instances 0 -}}
                  {{- $instance.Name -}}
                {{- end -}}
              </strong>
              {{- if $vgroup.Namespace }}
                 in namespace {{ $vgroup.Namespace | markupPlaceholders }}
              {{- end }}:
              {{ $vgroup.Message | markupPlaceholders }}
            </div>
            <div class="violation-instances {{ if gt (len $vgroup.Instances) 3 }}folded{{ end }}">
              {{ if gt (len $vgroup.Instances) 3 }}<div class="unfolder">{{ len $vgroup.Instances }} instances in total <a href="#">(expand)</a></div>{{ end }}
              {{- range $instance := $vgroup.Instances }}
                {{- $info := index $.ClusterInfos $instance.ClusterName }}
                <div class="violation-instance" data-layer="{{ $info.Layer }}" data-type="{{ $info.Type }}">{{ $instance.ClusterName }}: {{ $instance.Name }}</div>
              {{- end }}
            </div>
          </li>
        {{- end }}
      </ul>
    </section>
  {{- end }}

</main>

<script src="/static/behavior.js"></script>
