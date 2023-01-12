<!DOCTYPE html>
<title>DOOP: Decentralized Observer of Policies</title>
<link rel="stylesheet" href="/static/style.css">

<div class="loading-overlay">
  Loading...
</div>

<header>
  <h1>Decentralized Observer of Policies</h1>
  <form>
    <select name="layer">
      <option value="all" selected>All regions</option>
      {{- range $layer := $.AllClusterLayers }}
        <option value="{{ $layer }}">{{ $layer | titlecase }}</option>
      {{- end }}
    </select>
    <select name="type">
      <option value="all" selected>All clusters</option>
      {{- range $type := $.AllClusterTypes }}
        <option value="{{ $type }}">{{ $type | titlecase }}</option>
      {{- end }}
    </select>
    <select name="supportGroup">
      <option value="all" selected>All support groups</option>
      <option value="none">No support group</option>
      {{- range $support_group := $.AllSupportGroups }}
        {{- if ne $support_group "none" }}
          <option value="{{ $support_group }}">{{ $support_group }}</option>
        {{- end }}
      {{- end }}
    </select>
    <select name="service">
      <option value="all" selected>All services</option>
      <option value="none">No service</option>
      {{- range $service := $.AllServiceLabels }}
        {{- if ne $service "none" }}
          <option value="{{ $service }}">{{ $service }}</option>
        {{- end }}
      {{- end }}
    </select>
    <input name="search" id="search" autofocus type="text" placeholder="Search">
  </form>
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
          {{- $info := index $.APIData.ClusterInfos $cluster }}
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
        {{- range $vgroup := index $.APIData.ViolationGroups $kind }}
          <li>
            <div class="violation-details">
              {{- if ne $vgroup.SupportGroupLabel "none" -}}
                <span class="support-labels">
                  <span class="support-group">{{ $vgroup.SupportGroupLabel }}</span>
                  {{- if ne $vgroup.ServiceLabel "none" -}}<span class="service">{{ $vgroup.ServiceLabel }}</span>{{- end -}}
                </span>
              {{- end }}
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
                {{- $info := index $.APIData.ClusterInfos $instance.ClusterName }}
                <div class="violation-instance" data-layer="{{ $info.Layer }}" data-type="{{ $info.Type }}" data-support-group="{{ $vgroup.SupportGroupLabel }}" data-service="{{ $vgroup.ServiceLabel }}">{{ $instance.ClusterName }}: {{ $instance.Name }}</div>
              {{- end }}
            </div>
          </li>
        {{- end }}
      </ul>
    </section>
  {{- end }}

</main>

<script src="/static/behavior.js"></script>
