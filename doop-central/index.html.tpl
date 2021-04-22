<!DOCTYPE html>
<link rel="stylesheet" href="/static/style.css">
<h1>
  Decentralized Observer Of Policies
  <small><a href="https://futurama.fandom.com/wiki/Democratic_Order_of_Planets">[huh?]</a></small>
</h1>

<h2>Cluster health</h2>

<table>
  <thead>
    {{- range $clusterGroup := $.AllClusterGroups }}
      <tr>
        <th class="nobr">&nbsp;</th>
        {{- range $cluster := index $.ClustersByGroup $clusterGroup }}
          <th class="nobr"><div>{{ $cluster }}</div></th>
        {{- end }}
      </tr>
      <tr>
        <th class="nobr">Oldest data</th>
        {{- range $cluster := index $.ClustersByGroup $clusterGroup }}
          {{- $info := index $.ClusterInfos $cluster }}
          <td class="nobr center {{ $info.OldestAuditCSSClass }}">{{ printf "%.1fs" $info.OldestAuditAgeSecs }}</th>
        {{- end }}
      </tr>
      <tr>
        <th class="nobr">Newest data</th>
        {{- range $cluster := index $.ClustersByGroup $clusterGroup }}
          {{- $info := index $.ClusterInfos $cluster }}
          <td class="nobr center {{ $info.NewestAuditCSSClass }}">{{ printf "%.1fs" $info.NewestAuditAgeSecs }}</th>
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
