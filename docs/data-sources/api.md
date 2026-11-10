---
page_title: "konnectextra_api Data Source"
description: |-
  Look up a catalog API by name and version.
---

# konnectextra_api (Data Source)

```terraform
data "konnectextra_api" "example" {
  name    = "Payments"
  version = "v1"
}
```

Both `name` and `version` are required. Computed attributes are `id`, `description`, `slug`, `created_at`, and `updated_at`.
