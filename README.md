# oauth-static-webserver

## Status

**IMPORTANT:** **Do not use this software in any Production Environment.**
This project is intended for **testing and educational purposes only**.

The software is **not fully tested** yet and is currently under active development. There are no immediate plans to make it production-ready.

## Goals

- easy to use static web server with OIDC Protection
- simple configuration with one yaml file and some env Variables
- extensive logging

## TODOs

- [x] Integrate **TLS** for use without a reverse proxy.
- [ ] Implement **extensive testing** (unit and integration tests).
  - [x] basic Unit tests for core components.
- [x] Introduce **fine-grained access control rules** (beyond simple group membership).
- [ ] **Documentation:**
    - [x] Fully cover **Usage and Installation methods**.
    - [ ] Complete **code documentation** (GoDoc).
    - [ ] Improve readability and helpfulness of the **README**.
    - [ ] Conduct a language check (grammar, syntax, and style).
- [ ] Integrate a **Prometheus interface** for exporting monitoring data.
- [ ] Implement a **more robust and flexible IdP integration**:
    - [ ] Add native OAuth2 support for major providers like Google, Microsoft, etc.
- [x] Add a **Licence**.

---

## Installation

The simplest method is to use the included **Docker Compose**, which automatically build the required image.

### manual

Simply build the project with:
```
go build -o oauth-static-webserver
```

Then you can execute the binary.

### docker

At this time there is **no official image available**, you must build it by your  self.

I recommend using the provided Docker Compose file to start the service:
```
docker compose -f compose-build.yaml up
```

**Additional Flags:**
- `-d`: Run in detached mode (background).
- `--build`: Force the image to be rebuilt.

---

## Usage

### Environment Variables and Configuration

First, you must provide the required environment variables for OIDC:
- `LOG_LEVEL`: The logging level (e.g., `debug`, `info`, `warn`, `error`). Default is `info`.
- `HOST_ADDRESS`: The host address to bind the server (e.g., `localhost`).
- `HOST_PORT`: The host and port to bind the server (e.g., `1235`). Default is `8080`.
- `SESSION_KEY`: A secret key for session management (e.g., `mysecretkey123`). **This is required for secure session handling.**
- `SESSION_STORE_PROVIDER`: The session store provider (e.g., `filesystem`, `redis`). Default is `filesystem`.
- `SESSION_STORE_DIRECTORY`: The directory for session storage when using `filesystem` (e.g., `/tmp/sessions`).
- `SESSION_STORE_REDIS_ADDRESS`: The Redis server address when using `redis` (e.g., `localhost`).
- `SESSION_STORE_REDIS_PORT`: The Redis server port when using `redis` (e.g., `6379`). Default is `6379`.
- `SESSION_STORE_REDIS_USERNAME`: The Redis server username when using `redis` (e.g., `default`).
- `SESSION_STORE_REDIS_PASSWORD`: The Redis server password when using `redis` (e.g., `mypassword`).
- `SESSION_STORE_REDIS_DB`: The Redis database number when using `redis` (e.g., `0`). Default is `0`.
- `SESSION_STORE_REDIS_POOL_SIZE`: The Redis connection pool size when using `redis` (e.g., `10`). Default is `10`.
- `CONFIG_PATH`: The path to the configuration file (e.g., `./config.yaml`). Default is `/etc/oauth-static-webserver/config.yaml`.

All environment variables are required except those with default values.
The requirement for the `SESSION_STORE_REDIS_*` and `SESSION_STORE_DIRECTORY` variables depends on the chosen `SESSION_STORE_PROVIDER`.

### Configuration File

Also, you must configure the server using a `config.yaml` file:
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
    dir: "/var/www/page1"
    url: "/static/page1"
  - id: page2
    dir: "/var/www/page2"
    url: "/static/page2"
    protection:
      provider: idp
      groups:
        - static_access
```

You will first define the available provider with:
- `id` - The reference id inside the webserver
- `config_url` - The well-known url from you OIDC Application for the IdP
- `client_id` - The client id from the Application
- `client_secret` - The client secret from the Application

The `base_url` are the publicly exposed base url.
Typically, this contains the url to your reverse proxy and maybe the subpath, which proxies to this static webserver. 

Then you define the static pages:
- `id` - The reference id of the static page (only for logging required)
- `dir` - The path to the source directory. All content will be available from the `url`.
- `url` - The root path to the static page. (serves the content from `dir`)
- `protection` - The protection settings (optional)

When no `protection` will be provided, then all content is publicly available.
The `protection` contains:
- `provider` - The `id` of the used OIDC Provider
- `groups` - A list of all groups which are allowed to access the content. When no entry will be defined, then all authenticated user will be able to access the content.
