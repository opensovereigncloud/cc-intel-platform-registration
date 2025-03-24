{{/*
Expand the name of the chart.
*/}}
{{- define "cc-intel-platform-registration.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cc-intel-platform-registration.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "cc-intel-platform-registration.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "cc-intel-platform-registration.labels" -}}
helm.sh/chart: {{ include "cc-intel-platform-registration.chart" . }}
{{ include "cc-intel-platform-registration.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "cc-intel-platform-registration.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cc-intel-platform-registration.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "cc-intel-platform-registration.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "cc-intel-platform-registration.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}



{{- define "validate.logLevel" -}}
  {{- $validValues := list "debug" "info" "warn" "error" -}}
  {{- if not (has . $validValues) -}}
    {{- fail (printf "Invalid log-level: %s. Must be one of: %v" . $validValues) -}}
  {{- end -}}
{{- end -}}

{{- define "validate.encoder" -}}
  {{- $validValues := list "json" "console" -}}
  {{- if not (has . $validValues) -}}
    {{- fail (printf "Invalid encoder: %s. Must be one of: %v" . $validValues) -}}
  {{- end -}}
{{- end -}}

{{- define "validate.timeEncoding" -}}
  {{- $validValues := list "rfc3339" "rfc3339nano" "iso8601" "millis" "nanos" -}}
  {{- if not (has . $validValues) -}}
    {{- fail (printf "Invalid time-encoding: %s. Must be one of: %v" . $validValues) -}}
  {{- end -}}
{{- end -}}


{{- define "validate.interval" -}}
  {{- $value := . | int -}}  # Convert to integer
  {{- if or (lt $value 1) (ne (printf "%v" $value) (printf "%v" .)) -}}
    {{- fail (printf "Invalid intervalInMinutes: %v. Must be a non-zero positive number." .) -}}
  {{- end -}}
{{- end -}}
