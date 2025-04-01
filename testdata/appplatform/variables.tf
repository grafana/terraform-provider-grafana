variable "grafana_auth" {
  type        = string
  description = "[REQUIRED] Grafana instance service account token. Should have privileges for reading & writing dashboards & folders."
}

variable "grafana_url" {
  type        = string
  description = "[REQUIRED] Grafana instance URL."
  default     = "https://localhost:3000/"
}

variable "grafana_tls_cert" {
  type        = string
  description = "[OPTIONAL] Grafana client certificate."
  default     = ""
}

variable "grafana_tls_key" {
  type        = string
  description = "[OPTIONAL] Grafana client certificate key."
  default     = ""
}

variable "grafana_tls_ca" {
  type        = string
  description = "[OPTIONAL] Grafana CA certificate."
  default     = ""
}
