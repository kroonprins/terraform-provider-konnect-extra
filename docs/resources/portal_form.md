---
page_title: "konnectextra_portal_form Resource"
description: |-
  Manage a custom developer portal form.
---

# konnectextra_portal_form (Resource)

Manage a custom developer portal form. The `fields_json` value follows the [Konnect Create Form API](https://developer.konghq.com/api/konnect/portal-management/v3/#/operations/create-portal-form), including the order of the fields. Use `jsonencode` to create the JSON array.

## Example Usage

```terraform
resource "konnectextra_portal_form" "example" {
  portal_id = konnect_portal.example.id
  name      = "api-registration"
  type      = "api_registration"
  status    = "unpublished"
  fields_json = jsonencode([
    { name = "api_id", type = "text", label = "API" },
    { name = "submit", type = "submit", value = "Register" }
  ])
}
```

Every form needs exactly one `submit` field. `developer_registration` forms also require `full_name` (text) and `email` (email) fields and must be named `developer-registration`; `api_registration` forms require `api_id` (text). Changing `portal_id` or `type` replaces the form. Deleting a form unlinks any API publications that use it, but previously submitted responses are retained by Konnect.

## Schema

### Required

- `portal_id` (String) The ID of the portal. Changing this replaces the form.
- `name` (String) The form name.
- `type` (String) `developer_registration` or `api_registration`. Changing this replaces the form.
- `status` (String) `published` or `unpublished`.
- `fields_json` (String) JSON array of form fields in display order.

### Read-Only

- `id` (String) The ID of the form.
- `created_at` (String) An ISO-8601 timestamp of when the form was created.
- `updated_at` (String) An ISO-8601 timestamp of when the form was last updated.

## Migrating to the Official Provider

Once an official resource supports import, remove this resource from state **without destroying the remote form**, configure the official resource, and import the same form using the official provider's required ID format. Confirm the official field schema and import format before making the switch. A `moved` block across resource types is possible only if the official provider explicitly implements support for this source resource.
