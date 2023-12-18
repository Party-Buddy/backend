# Party Buddy backend
This repository hosts the source code of the backend.

## Running
The easiest way to get started is via docker-compose.

```
$ make docker
```

If you're a fan of Podman, set the compose executable accordingly as follows:

```
$ make DOCKER_COMPOSE=podman-compose docker
```

### The default password
The Makefile sets the DB user password to zxcvbnM1.
If you'd like a bit more safety, write your password into the `./container/postgres-passwd.txt` file before running `make docker` for the first time.

### Rebuilding
You can rebuild the images by running

```
$ make docker-build
```

### Ports
The containers expose two ports:

- 9601: PostgreSQL
- 9602: the backend HTTP server

### Volumes
The following volumes are defined:

- `pb-db`: the PostgreSQL DB files
- `pb-backend-cfg`: backend config files
- `pb-backend-img`: user image files (uploaded via the API)
