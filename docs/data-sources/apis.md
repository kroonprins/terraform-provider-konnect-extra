---
page_title: "konnectextra_apis Data Source"
description: |-
  List catalog APIs by name.
---

# konnectextra_apis (Data Source)

```terraform
data "konnectextra_apis" "example" {
  name = "Payments"
}
```

`name` is required. `apis` is a computed list of matching APIs, each with `id`, `name`, `version`, `description`, `slug`, `created_at`, and `updated_at`.
