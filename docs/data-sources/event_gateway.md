---
page_title: "konnectextra_event_gateway Data Source"
description: |-
  Look up an Event Gateway by name.
---

# konnectextra_event_gateway (Data Source)

```terraform
data "konnectextra_event_gateway" "example" {
  name = "my-event-gateway"
}
```

The lookup fails unless exactly one gateway matches.
