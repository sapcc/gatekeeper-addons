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

  {{- range $kind := $.AllTemplateKinds }}
    <h2>Constraint: {{ $kind }}</h2>

    <ul>
      {{- range $vgroup := index $.ViolationGroups $kind }}
        <li>
          {{- if $vgroup.Namespace }}
            <strong>{{ $vgroup.Kind }}</strong> in namespace {{ $vgroup.Namespace }}:
          {{- else }}
            {{ $vgroup.Kind }}:
          {{- end }}
          {{- if gt (len $vgroup.Instances) 1 }}
            <strong>{{ $vgroup.NamePattern }}</strong>
          {{- else }}
            {{- $instance := index $vgroup.Instances 0 }}
            <strong>{{ $instance.Name }}</strong>
          {{- end }}
          <div class="violation-details">
            <code>{{ $vgroup.Message }}</code>
            {{- range $instance := $vgroup.Instances }}
              <br><small>{{ $instance.ClusterName }}: {{ $instance.Name }}</small>
            {{- end }}
          </div>
        </li>
      {{- end }}
    </ul>
  {{- end }}

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

</main>
