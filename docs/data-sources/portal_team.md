---
page_title: "konnectextra_portal_team Data Source"
description: |-
  Look up a portal team by portal ID and team name.
---

# konnectextra_portal_team (Data Source)

Look up a portal team by portal ID and team name.

## Example Usage

```terraform
data "konnectextra_portal_team" "example" {
  portal_id = "d32d905a-ed33-46a3-a093-d8f536af9a8a"
  name      = "My Team"
}
```

## Schema

### Required

- `portal_id` (String) The ID of the portal.
- `name` (String) The name of the team to look up.

### Read-Only

- `id` (String) The ID of the team.
- `description` (String) The description of the team.
- `created_at` (String) An ISO-8601 timestamp of when the team was created.
- `updated_at` (String) An ISO-8601 timestamp of when the team was last updated.
- `can_own_applications` (Boolean) Whether the team is allowed to own applications.
- `konnect_managed` (Boolean) Whether the team's membership is managed by Konnect instead of being synced from an identity provider's team mappings.
