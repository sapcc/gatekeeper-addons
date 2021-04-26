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

  {{- range $kind := $.AllTemplateKinds }}
    <section>
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
                  {{- $vgroup.NamePattern -}}
                {{- else -}}
                  {{- $instance := index $vgroup.Instances 0 -}}
                  {{- $instance.Name -}}
                {{- end -}}
              </strong>
              {{- if $vgroup.Namespace }}
                 in namespace {{ $vgroup.Namespace }}
              {{- end }}:
              {{ $vgroup.Message }}
            </div>
            <div class="violation-instances {{ if gt (len $vgroup.Instances) 3 }}folded{{ end }}">
              <div class="unfolder">{{ len $vgroup.Instances }} instances in total <a href="#">(expand)</a></div>
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

  <section class="folded">
    <h2>Gatekeeper stats</h2>

    <table>
      <thead>
        {{- range $ctype := $.AllClusterTypes }}
          <tr>
            <th class="nobr">&nbsp;</th>
            {{- range $cluster := $.AllClusters }}
              {{- $info := index $.ClusterInfos $cluster }}
              {{- if eq $info.Type $ctype }}
                <th class="nobr"><div>{{ $cluster }}</div></th>
              {{- end }}
            {{- end }}
          </tr>
          <tr>
            <th class="nobr">Oldest data</th>
            {{- range $cluster := $.AllClusters }}
              {{- $info := index $.ClusterInfos $cluster }}
              {{- if eq $info.Type $ctype }}
                <td class="nobr center {{ $info.OldestAuditCSSClass }}">{{ printf "%.1fs" $info.OldestAuditAgeSecs }}</th>
              {{- end }}
            {{- end }}
          </tr>
          <tr>
            <th class="nobr">Newest data</th>
            {{- range $cluster := $.AllClusters }}
              {{- $info := index $.ClusterInfos $cluster }}
              {{- if eq $info.Type $ctype }}
                <td class="nobr center {{ $info.NewestAuditCSSClass }}">{{ printf "%.1fs" $info.NewestAuditAgeSecs }}</th>
              {{- end }}
            {{- end }}
          </tr>
        {{- end }}
      </thead>
    </table>
  </section>

</main>

<script src="/static/behavior.js"></script>
