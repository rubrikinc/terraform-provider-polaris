# Using SLA domain ID.
data "polaris_sla_domain" "sla_domain" {
  id = "3c1a891a-340c-4b8a-a1ca-adec4d5914e4"
}

# Using SLA domain name.
data "polaris_sla_domain" "sla_domain" {
  name = "bronze"
}
