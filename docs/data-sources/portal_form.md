---
page_title: "konnectextra_portal_form Data Source"
description: |-
  Look up a developer portal form by portal ID and name.
---

# konnectextra_portal_form (Data Source)

Look up a custom developer portal form by portal ID and name. The default `developer_registration` form is not returned by the API unless it has been explicitly created.

## Example Usage

```terraform
data "konnectextra_portal_form" "example" {
  portal_id = "d32d905a-ed33-46a3-a093-d8f536af9a8a"
  name      = "api-registration"
}
```

## Schema

### Required

- `portal_id` (String) The ID of the portal.
- `name` (String) The name of the form to look up.

### Read-Only

- `id` (String) The ID of the form.
- `type` (String) The form type (developer_registration or api_registration).
- `status` (String) The publication status of the form.
- `created_at` (String) An ISO-8601 timestamp of when the form was created.
- `updated_at` (String) An ISO-8601 timestamp of when the form was last updated.
