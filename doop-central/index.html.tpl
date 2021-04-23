<!DOCTYPE html>
<link rel="stylesheet" href="/static/style.css">

<header>
  <h1>Decentralized Observer of Policies</h1>
  <div class="buttons">
    {{- range $layer := $.AllClusterLayers }}
      <div class="selected">{{ $layer | titlecase }}</div>
    {{- end }}
  </div>
  <div class="buttons">
    {{- range $type := $.AllClusterTypes }}
      <div class="selected">{{ $type | titlecase }}</div>
    {{- end }}
  </div>
  <input type="text" placeholder="Search">
</header>

<main>

  <h2>Cluster health</h2>

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

  <!-- TODO: remove debug display -->
  {{- range $cluster := $.AllClusters }}
    <h2>Raw report: {{ $cluster }}</h2>
    <pre><code>{{ index $.Reports $cluster | jsonIndent }}</code></pre>
  {{- end }}

</main>
