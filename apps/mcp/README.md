# paca-mcp — the MCP server of PACA

`paca-mcp` is the MCP server of PACA: the process an AI agent talks to. It
serves the MCP surface over Streamable HTTP, from the official MCP Go SDK, and
it serves the same domain code the CLI composes directly. The two binaries are
independent composition roots over the shared core (`packages/core`); the CLI
never calls this server.

The server carries no tool yet. The tools arrive in the sessions that follow:
the contract layer first, then the tool sessions.

## Running it

```
paca-mcp --db ./paca.db
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--bind` | `127.0.0.1` | address the server listens on |
| `--port` | `7333` | TCP port the server listens on |
| `--auth-token` | — | bearer token the MCP surface demands |
| `--tls-cert` | — | path of the TLS certificate file |
| `--tls-key` | — | path of the TLS private key file |
| `--db` | — | path of the SQLite file the server serves (required) |
| `--require-auth` | `false` | refuse to start without a bearer token |

The database is migrated at startup: the runner of the shared core applies only
the migrations the file has not applied yet, so an already migrated file is
untouched and a new file is created with the schema.

## The rules of the startup

A server bound to an address that is not loopback is exposed to the network,
so it refuses to start unless it has both a bearer token and the TLS
material:

- a loopback address may run plain HTTP, and the token is optional unless
  `--require-auth` is set;
- any other address needs `--auth-token`, `--tls-cert` and `--tls-key`;
- `--require-auth` without a token is a refusal too, on any address.

The refusal names the reason and never carries the token. The token is
checked on every request of the MCP surface, with a constant-time comparison,
and it never reaches a log line.

## The endpoints

| Path | What it answers |
| --- | --- |
| `/mcp` | the MCP surface, over Streamable HTTP |
| `/healthz` | liveness: `200` with the plain text `ok`, outside the MCP surface and outside the authentication |

## Deploying it on the local network

The deployment target of PACA is another machine of the same home network,
reached from the machine the agent runs on. That is a non-loopback address, so
the server needs a token and TLS. A self-signed certificate is enough for a
home network:

```
openssl req -x509 -newkey rsa:4096 -nodes -days 365 \
  -keyout key.pem -out cert.pem -subj "/CN=paca-mcp"

paca-mcp --db /var/lib/paca/paca.db --bind 0.0.0.0 --port 7333 \
  --auth-token "$PACA_TOKEN" --tls-cert cert.pem --tls-key key.pem
```

The client then calls `https://<host>:7333/mcp` with the header
`Authorization: Bearer <token>`.

## Tests

```
go test ./...
```
