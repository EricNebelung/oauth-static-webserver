---
title: Installation
---

Currently, there are two installation methods:

- manual installation
- using Docker

The usage of Docker is recommended and a prebuild image is provided via the Github Container Registry.

### Docker

Requirements:

- Docker with support for Docker Compose installed (or an alternative like Podman with Compose support)
- Knowledge of how to use Docker Compose
- Internet access to pull the image

You can pull the latest image from the Github Container Registry with the following command:
```bash
docker pull ghcr.io/ericnebelung/oauth-static-webserver:latest
```
With the following compose file you can start the service:
```yaml
networks:
  oauth-static-webserver:
    external: false

services:
  oauth-static-webserver:
    image: ghcr.io/ericnebelung/oauth-static-webserver:latest
    restart: always
    environment:
      - SESSION_KEY=some_secret_key
      - SESSION_STORE_DRIVER=redis
      - SESSION_REDIS_ADDRESS=session-storage
      - SESSION_REDIS_PORT=6379
    ports:
      - "8080:8080"
    networks:
      - oauth-static-webserver
    volumes:
      - "/etc/oauth-resource-proxy/config.yaml:/etc/oauth-resource-proxy/config.yaml:ro"
      - "/var/www:/var/www:ro"
    depends_on:
      - session-storage
  session-storage:
    image: redis:latest
    restart: always
    networks:
      - oauth-static-webserver
    volumes:
      - cache:/data

volumes:
  cache:
    driver: local
```

You must adjust some parameters like the `SESSION_KEY` and the volume mounts for your configuration file and static web content.
The `/var/www` are the default root directory for the static web content, which is also set as working directory in the image.
You can also add extra volume mounts and simply set the paths in the configuration file.

For more details about the configuration, please refer to the [Configuration](configuration.md) documentation.

### Manual

You can also build the project manually.
The following requirements must be fulfilled:

- Go 1.24 or higher (only tested with Go 1.24 on Linux)
- Git (to clone the repository)
- Internet access to download dependencies

To build the project, follow these steps:

1. Clone the repository:
   ```bash
   git clone https://github.com/EricNebelung/oauth-static-webserver
   ```
2. Change into the project directory:
   ```bash
   cd oauth-static-webserver
   ```
3. Build the project using Go:
   ```bash
   go build -o oauth-static-webserver
   ```
4. Add the executable permission to the binary:
   ```bash
   chmod +x oauth-static-webserver
   ```

Now an executable binary named `oauth-static-webserver` should be available in the project directory.
You can write your own systemd service or any other service configuration.
Also, you can run the binary directly from the command line.

The programm does not use any command line argument yet, but all configuration is done via environment variables and a configuration file.
For more details about the configuration, please refer to the [Configuration](configuration.md) documentation.
