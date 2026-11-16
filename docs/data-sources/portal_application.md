---
page_title: "konnectextra_portal_application Data Source"
description: |-
  Look up a portal application by portal ID and name.
---

# konnectextra_portal_application (Data Source)

```terraform
data "konnectextra_portal_application" "example" {
  portal_id = "d32d905a-ed33-46a3-a093-d8f536af9a8a"
  name      = "My Application"
  owner = {
    id   = "developer-or-team-id"
    type = "developer"
  }
}
```

`portal_id` and `name` are required. The optional `owner` matches the official `konnect_portal_application` owner structure and restricts the lookup by an owner `id` and `type` (`developer` or `team`). Computed attributes are `id`, `description`, `client_id`, `auth_strategy_id`, `registration_count`, `created_at`, and `updated_at`.
