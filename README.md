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

## Usage

All information about installation and configuration can be found in the documentation, hosted as Github Pages: https://ericnebelung.github.io/oauth-static-webserver/
