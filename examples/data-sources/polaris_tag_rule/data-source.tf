# Look up by tag rule ID.
data "polaris_tag_rule" "rule" {
  id = "59abc6bd-1baf-477e-8767-686e0c1d89ba"
}

# Look up by tag rule name.
data "polaris_tag_rule" "rule" {
  name = "my-tag-rule"
}
