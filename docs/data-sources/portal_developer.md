---
page_title: "konnectextra_portal_developer Data Source"
description: |-
  Look up a portal developer by portal ID and full name.
---

# konnectextra_portal_developer (Data Source)

```terraform
data "konnectextra_portal_developer" "example" {
  portal_id = "d32d905a-ed33-46a3-a093-d8f536af9a8a"
  name      = "Jane Doe"
}
```

`portal_id` and `name` are required. Computed attributes are `id`, `email`, `full_name`, `status`, `created_at`, and `updated_at`.
