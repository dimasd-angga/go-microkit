{{- define "go-microkit.image" -}}
{{- printf "%s/%s:%s" .Values.global.image.registry .service .Values.global.image.tag -}}
{{- end -}}

{{- define "go-microkit.labels" -}}
app.kubernetes.io/name: go-microkit
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: {{ .service }}
{{- end -}}
