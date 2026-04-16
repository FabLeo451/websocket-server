# Ekhoes Server

CLI tool to start and manage **Ekhoes Server**.

### Usage

```
ekhoes-server --help
```

### Available Commands

```
completion  Generate the autocompletion script for the specified shell
help        Help about any command
install     Create database and execute init script
start       Start server
```

### Command Execution

Example of running a command:

```
ekhoes-server [command]
```

### Global Flags

```
-h, --help     Show help for ekhoes-server
-l, --local    Use a local on-disk database
```

### More Information

To get help for a specific command:

```
ekhoes-server [command] --help
```
### Environment Variables

| Variable      | Description |
| -------------- | --------------------------------------------- |
| EKHOES_INSTANCE_NAME | Name of the instance shown in the root page |
| EKHOES_PORT | Port the server will listen to |
| EKHOES_DB_ENABLED | If true, server will connect to Postgres database at startup |
| EKHOES_DB_HOST | Database hostname or ip address |
| EKHOES_DB_PORT | Database port |
| EKHOES_DB_USER | Database user |
| EKHOES_DB_PASSWORD | Database password |
| EKHOES_DB_NAME | Database name |
| EKHOES_DB_SCHEMA | Database schema |
| EKHOES_DB_HEARTBEAT | Number of seconds between pings to database |
| EKHOES_DB_POOLSIZE | Database poolsize |
| EKHOES_REDIS_ENABLED | If true, server will connect to Redis database at startup |
| EKHOES_REDIS_HOST | Redis hostname or ip address |
| EKHOES_REDIS_PORT | Redis port |
| EKHOES_REDIS_PASSWORD | Redis password |
| EKHOES_REDIS_POOLSIZE | Redis poolsize |
| EKHOES_JWT_SECRET | String used to encode/decode JWT tokens |
| EKHOES_HOST_MOUNT_POINT | Mount point for host filesystem when running as a container |
| EKHOES_MODULES | Comma separated list of modules to be started |
