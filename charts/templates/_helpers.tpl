{{/*
Common labels
*/}}
{{- define "reservia-admin-api.labels" -}}
app.kubernetes.io/name: reservia-admin-api
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}