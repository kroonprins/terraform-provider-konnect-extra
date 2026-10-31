---
page_title: "konnect-extra Provider"
description: |-
  A supplemental Terraform provider for Kong Konnect, providing data sources and resources not yet available in the official provider.
---

# konnect-extra Provider

A supplemental provider for [Kong Konnect](https://konghq.com/products/kong-konnect) that provides data sources and resources not yet available in the [official provider](https://registry.terraform.io/providers/Kong/konnect/latest).

This provider is intended to be temporary. As the official `Kong/konnect` provider adds support for these features, the corresponding resources here will be deprecated and removed.

## Example Usage

```terraform
terraform {
  required_providers {
    konnectextra = {
      source = "kroonprins/konnect-extra"
    }
  }
}

provider "konnectextra" {
  # Configuration options below, or use environment variables
}
```

## Authentication

Authentication uses the same environment variables as the official Kong Konnect provider:

| Attribute                      | Environment Variable  |
|-------------------------------|-----------------------|
| `personal_access_token`        | `KONNECT_TOKEN`       |
| `system_account_access_token`  | `KONNECT_SPAT`        |
| `server_url`                   | `KONNECT_SERVER_URL`  |

## Schema

### Optional

- `personal_access_token` (String, Sensitive) A personal access token for authenticating with the Konnect API. Configurable via environment variable `KONNECT_TOKEN`.
- `system_account_access_token` (String, Sensitive) A system account access token for authenticating with the Konnect API. Configurable via environment variable `KONNECT_SPAT`.
- `server_url` (String) The Konnect API server URL. Defaults to `https://global.api.konghq.com`. Configurable via environment variable `KONNECT_SERVER_URL`.
