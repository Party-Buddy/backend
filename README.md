# Party Buddy backend
This repository hosts the source code of the backend.

## Running
The easiest way to get started is via the compose.yml file.
Setup steps for Podman:

```
$ echo "my-pg-password" | podman secret create postgres-passwd
$ podman compose -f compose.yaml up
```

(If you're stuck with Docker, just replace all uses of `podman` with `docker`.)

### Ports
The containers expose two ports:

- 9601: PostgreSQL
- 9602: the backend HTTP server

### Volumes
The following volumes are defined:

- `pb-db`: the PostgreSQL DB files
- `pb-backend-cfg`: backend config files
- `pb-backend-img`: user image files (uploaded via the API)
