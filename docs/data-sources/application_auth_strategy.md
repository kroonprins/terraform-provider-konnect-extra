---
page_title: "konnectextra_application_auth_strategy Data Source"
description: |-
  Look up an application auth strategy by name.
---

# konnectextra_application_auth_strategy (Data Source)

Look up an application auth strategy by its Konnect name, not its portal display name.

## Example Usage

```terraform
data "konnectextra_application_auth_strategy" "example" {
  name = "My Auth Strategy"
}
```

## Schema

### Required

- `name` (String) The name of the application auth strategy.

### Read-Only

- `id` (String) The ID of the application auth strategy.
- `display_name` (String) The display name shown in the portal.
- `strategy_type` (String) The strategy type (key_auth or openid_connect).
- `active` (Boolean) Whether a published entity uses this strategy.
- `dcr_provider_id` (String) The ID of the associated DCR provider, if any.
- `supports_multiple_credentials` (Boolean) Whether this strategy supports multiple credentials.
- `created_at` (String) An ISO-8601 timestamp of when the strategy was created.
- `updated_at` (String) An ISO-8601 timestamp of when the strategy was last updated.
