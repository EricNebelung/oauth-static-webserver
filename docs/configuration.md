---
title: Configuration
---

The configuration of the OAuth Static Webserver is split into two main parts:

- Server Configuration to set up the server environment and session management.
- Site Configuration to define the static sites to be served and their respective OIDC protection settings.

## Server Configuration

The server configuration is done with environment variables.

| Field                          | Default                                         | Description                                                           |
|:-------------------------------|-------------------------------------------------|-----------------------------------------------------------------------|
| `LOG_LEVEL`                    | `info`                                          | The logging level like `debug`, `info`, `warn` or `error`             |
| `CONFIG_PATH`                  | `/etc/oauth-static-webserver/config.yaml`       | The path to the configuration file.                                   |
| `HOST_ADDRESS`                 |                                                 |                                                                       |
| `HOST_PORT`                    | `8080`                                          | The http(s) listen port.                                              |
| `HTTP2_MAX_CONCURRENT_STREAMS` | `100`                                           |                                                                       |
| `HTTP2_MAX_READ_FRAME_SIZE`    | `1048576` (1 KiB)                               |                                                                       |
| `HTTP2_IDLE_TIMEOUT`           | `10` (Seconds)                                  |                                                                       |
| `TLS_ENABLED`                  | `false`                                         | Enables the TLS functionality of the server.                          |
| `TLS_HTTP_REDIRECT`            | `true`                                          | Force HTTPS instead of HTTP                                           |
| `TLS_CERT_FILE`                |                                                 | The path to the TLS certificate file.                                 |
| `TLS_KEY_FILE`                 |                                                 | The path to the TLS key file.                                         |
| `TLS_AUTO_TLS`                 | `false`                                         | To use automatic TLS certificate request (Let's Encrypt)              |
| `TLS_AUTO_TLS_CERT_CACHE_DIR`  | Uses a tmp directory, when no path is provided. | The cert cache dir, required to prevent Let's Encrypt rate limiting.  |
| `SESSION_KEY`                  |                                                 | Session Encryption Key, should be a secret.                           |
| `SESSION_STORE_DRIVER`         | `filesystem`                                    | The session storage. Use `redis` to use a Redis DB.                   |
| `SESSION_STORE_DIRECTORY`      |                                                 | Path to the session storage (only required when `filesystem` is used) |
| `SESSION_REDIS_ADDRESS`        |                                                 | The Address of the redis server                                       |
| `SESSION_REDIS_PORT`           | `6379`                                          | The Port of the redis server                                          |
| `SESSION_REDIS_USERNAME`       |                                                 | Username for the authentication                                       |
| `SESSION_REDIS_PASSWORD`       |                                                 | Password for the authentication                                       |
| `SESSION_REDIS_DB`             | `0`                                             | Redis DB Index                                                        |
| `SESSION_REDIS_POOL_SIZE`      | `10`                                            | Connection pool size for the Redis DB                                 |

## Site Configuration

The site configuration is done with a YAML configuration file.

```yaml
oidc:
  # Base URL for callback (and more) - must accessible from the internet
  # often the reverse proxy
  base_url: "http://localhost:8080/"
  providers:
    - id: idp
      config_url: "[WELL-KNOWN-URL]"
      client_id: "[CLIENT_ID]"
      client_secret: "[CLIENT_SECRET_KEY]"
static_pages:
  - id: page1
    dir: "page1"
    url: "/static/page1"
  - id: page2
    dir: "/var/www/page2"
    url: "/static/page2"
    protection:
      provider: idp
      expression: "text.has_prefix(user.email, '@example.com') || user.level >= 3"
      groups:
        - group1
        - group2 
```

The example above shows the basic structure of the configuration file with all existing options.

Some explanations about the fields:

- `oidc.base_url`: The base URL where the server is reachable from the internet. This is often the URL of the reverse proxy in front of the server.
- `oidc.providers`: A list of OIDC providers to use for authentication.
  - `id`: A unique identifier for the provider.
  - `config_url`: The well-known URL of the OIDC provider.
  - `client_id`: The client ID for the OIDC application.
  - `client_secret`: The client secret for the OIDC application.
- `static_pages`: A list of static pages to be served.
  - `id`: A unique identifier for the static page (used for logging).
  - `dir`: The directory where the static content is located.
  - `url`: The URL path where the static content will be accessible.
  - `protection`: (Optional) The protection settings for the static page.
    - `provider`: The `id` of the OIDC provider to use for authentication.
    - `groups`: (Optional) A list of groups that are allowed to access the static page. If not specified, the group check is skipped.
    - `expression`: (Optional) A custom expression to evaluate for access control. The expression can use user attributes like `user.email`, `user.level`, etc.

The `protection` section is optional. If it is not provided, the static page will be publicly accessible without authentication.
Also, both `groups` and `expression` are optional inside the `protection`.
The expression will be evaluated first and then the group check.

The expression language is a fully featured programming language, but here it will be inserted into a boolean context. 
The language named "tengo" is documented here: https://github.com/d5/tengo/

Also for the expression context, the following stdlib modules are available:

- `text`: https://github.com/d5/tengo/blob/master/docs/stdlib-text.md
- `math`: https://github.com/d5/tengo/blob/master/docs/stdlib-math.md
- `times`: https://github.com/d5/tengo/blob/master/docs/stdlib-times.md

To use it, write the name first and then call the methods, e.g. `text.has_prefix(user.email, '@example.com')`.
All user attributes are available via the `user` variable, e.g. `user.email`, `user.name`, `user.level`, etc.
