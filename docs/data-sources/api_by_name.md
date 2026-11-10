---
page_title: "konnectextra_api_by_name Data Source"
description: |-
  Look up exactly one catalog API by name.
---

# konnectextra_api_by_name (Data Source)

```terraform
data "konnectextra_api_by_name" "example" {
  name = "Payments"
}
```

`name` is required. The lookup fails if no API or more than one API matches. Computed attributes are `id`, `version`, `description`, `slug`, `created_at`, and `updated_at`.
