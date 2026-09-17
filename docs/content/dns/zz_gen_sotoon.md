---
title: "Sotoon"
date: 2019-03-03T16:39:46+01:00
draft: false
slug: sotoon
dnsprovider:
  since:    "v5.5.0"
  code:     "sotoon"
  url:      "https://sotoon.ir"
---

<!-- THIS DOCUMENTATION IS AUTO-GENERATED. PLEASE DO NOT EDIT. -->
<!-- providers/dns/sotoon/sotoon.toml -->
<!-- THIS DOCUMENTATION IS AUTO-GENERATED. PLEASE DO NOT EDIT. -->


Configuration for [Sotoon](https://sotoon.ir).


<!--more-->

- Code: `sotoon`
- Since: v5.5.0


Here is an example bash command using the Sotoon provider:

```bash
SOTOON_TOKEN=xxxxx \
SOTOON_WORKSPACE_UUID=xxxxx \
lego run --dns sotoon -d '*.example.com' -d example.com
```




## Credentials

| Environment Variable Name | Description |
|-----------------------|-------------|
| `SOTOON_TOKEN` | API token |
| `SOTOON_WORKSPACE_UUID` | Workspace UUID |

The environment variable names can be suffixed by `_FILE` to reference a file instead of a value.
More information [here]({{% ref "dns#configuration-and-credentials" %}}).


## Additional Configuration

| Environment Variable Name | Description |
|--------------------------------|-------------|
| `SOTOON_HTTP_TIMEOUT` | API request timeout in seconds (Default: 30) |
| `SOTOON_POLLING_INTERVAL` | Time between DNS propagation check in seconds (Default: 2) |
| `SOTOON_PROPAGATION_TIMEOUT` | Maximum waiting time for DNS propagation in seconds (Default: 120) |
| `SOTOON_SEQUENCE_INTERVAL` | Time between sequential requests in seconds (Default: 1) |
| `SOTOON_TTL` | The TTL of the TXT record used for the DNS challenge in seconds (Default: 120) |

The environment variable names can be suffixed by `_FILE` to reference a file instead of a value.
More information [here]({{% ref "dns#configuration-and-credentials" %}}).

Get your API token at https://ocean.sotoon.ir/profile/tokens

Get your workspace UUID at https://ocean.sotoon.ir/profile/workspaces



## More information

- [API documentation](https://docs.sotoon.ir/products/cdn/api-reference/api-getting-started)

<!-- THIS DOCUMENTATION IS AUTO-GENERATED. PLEASE DO NOT EDIT. -->
<!-- providers/dns/sotoon/sotoon.toml -->
<!-- THIS DOCUMENTATION IS AUTO-GENERATED. PLEASE DO NOT EDIT. -->
