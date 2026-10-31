# terraform-provider-konnect-extra

A supplemental Terraform provider for [Kong Konnect](https://konghq.com/products/kong-konnect) that provides data sources and resources not yet available in the [official provider](https://registry.terraform.io/providers/Kong/konnect/latest).

**This provider is intended to be temporary.** As the official `Kong/konnect` provider adds support for these features, the corresponding resources here will be deprecated and removed. The goal is to unblock usage until the official provider catches up.

## Data Sources

### `konnectextra_portal_team`

Look up a portal team by portal ID and team name.

```hcl
data "konnectextra_portal_team" "example" {
  portal_id = "your-portal-id"
  name      = "My Team"
}
```

#### Arguments

- `portal_id` (Required) - The ID of the portal.
- `name` (Required) - The name of the team to look up.

#### Attributes

- `id` - The ID of the team.
- `description` - The description of the team.
- `created_at` - ISO-8601 timestamp of when the team was created.
- `updated_at` - ISO-8601 timestamp of when the team was last updated.
- `can_own_applications` - Whether the team is allowed to own applications.
- `konnect_managed` - Whether the team's membership is managed by Konnect.

### `konnectextra_application_auth_strategy`

Look up an application auth strategy by its Konnect name (not its portal display name).

```hcl
data "konnectextra_application_auth_strategy" "example" {
  name = "My Auth Strategy"
}
```

The computed `id` can be used with resources in the official provider. Other computed attributes are `display_name`, `strategy_type`, `active`, `dcr_provider_id` (null if absent), `supports_multiple_credentials`, `created_at`, and `updated_at`.

### `konnectextra_portal_form`

Look up a custom developer portal form by portal ID and name.

```hcl
data "konnectextra_portal_form" "example" {
  portal_id = "your-portal-id"
  name      = "api-registration"
}
```

Computed attributes are `id`, `type`, `status`, `created_at`, and `updated_at`. The default `developer_registration` form is only available from this lookup after it has been explicitly created in Konnect.

## Provider Configuration

```hcl
provider "konnectextra" {
  server_url            = "https://us.api.konghq.com"  # optional
  personal_access_token = "kpat_..."                   # optional if using env var
}
```

All configuration can also be set via environment variables, matching the official provider:

| Attribute                      | Environment Variable   |
|-------------------------------|----------------------|
| `personal_access_token`        | `KONNECT_TOKEN`       |
| `system_account_access_token`  | `KONNECT_SPAT`        |
| `server_url`                   | `KONNECT_SERVER_URL`  |

## Usage with the Official Provider

This provider is designed to be used alongside the official Kong Konnect provider:

```hcl
terraform {
  required_providers {
    konnect = {
      source = "Kong/konnect"
    }
    konnectextra = {
      source = "kroonprins/konnect-extra"
    }
  }
}

provider "konnect" {}

provider "konnectextra" {}

data "konnectextra_portal_team" "my_team" {
  portal_id = konnect_portal.my_portal.id
  name      = "My Team"
}
```
