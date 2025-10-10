{{- define "istio.common.setup" -}}
  {{- $global := .Values.global | default dict -}}
  {{- $customRegistry := "" -}}
  {{- if and $global.registry (ne $global.registry "docker.io") -}}
    {{- $customRegistry = $global.registry -}}
  {{- end -}}
  {{- dict "customRegistry" $customRegistry | toJson -}}
{{- end -}}

{{- define "istio.service.values" -}}
operator:
  enabled: false
rootCA:
  enabled: false
intermediateCAs:
  {{ `{{ .Cluster.metadata.name }}` }}:
    namespace: {{ `{{ .Cluster.metadata.namespace }}` }}
    certificate: false
    issuer: true
  kcm-cluster:
    certificate: false
    issuer: false
global:
{{- with .customRegistry }}
  hub: {{ . }}/istio
{{- end }}
  multiCluster:
    clusterName: {{ `{{ .Cluster.metadata.name }}` }}
  network: {{ `{{ .Cluster.metadata.name }}` }}-network
cert-manager-istio-csr:
{{- with .customRegistry }}
  image:
    repository: {{ . }}/jetstack/cert-manager-istio-csr
{{- end }}
  app:
    certmanager:
      issuer:
        name: k0rdent-istio-{{ `{{ .Cluster.metadata.namespace }}` }}-{{ `{{ .Cluster.metadata.name }}` }}-ca
    server:
      clusterID: {{ `{{ .Cluster.metadata.name }}` }}
{{- end -}}