# Pipeline Horn

GitLab pipeline failure alerts over a persistent websocket to a Raspberry Pi client that plays a sound.

## Architecture

1. GitLab group webhook calls `POST /webhook` on the server.
2. Server validates the GitLab token, filters failed pipelines for the configured group, and applies a 30 second cooldown.
3. Server sends a single-byte websocket binary frame (`0x01`) to the connected Pi client.
4. Pi client reconnects automatically and plays the configured sound file.

## Server

Copy [`.env.example`](.env.example) to `.env`, set secrets and `GITLAB_GROUP_PATH`, then start the published server image:

```bash
cp .env.example .env
docker compose up -d
```

Compose pulls `ghcr.io/frostwalk/pipeline-alert:latest` from the [pipeline-alert](https://github.com/FrostWalk/pipeline-alert) repository.

### Environment variables

| Variable | Purpose |
| --- | --- |
| `HOST` | Bind address, default `0.0.0.0` |
| `PORT` | Listen port, default `8080` |
| `WEBHOOK_SECRET` | Shared secret for GitLab webhook token header |
| `WEBSOCKET_SECRET` | Shared secret for Pi client websocket auth |
| `TOKEN_HEADER` | GitLab token header name, default `X-Gitlab-Token` |
| `GITLAB_GROUP_PATH` | GitLab group path used to scope failed pipeline alerts |

`config.json` can provide defaults for local development.

### GitLab webhook setup

1. Open the target GitLab group.
2. Create a group webhook for pipeline events.
3. Point it at `https://your-proxy.example/webhook`.
4. Set the secret token to the same value as `WEBHOOK_SECRET`.
5. Ensure failed pipelines in projects under `GITLAB_GROUP_PATH` are included.

## Client

### Run locally

```bash
go run ./cmd/client \
  --server_url=your-proxy.example \
  --server_port=443 \
  --websocket_secret=client-secret \
  --sound_path=./horn.mp3
```

Use `--accept_invalid_tls` only for local testing.

### Debian package

Download the latest `pipeline-horn-client` `.deb` from the [GitHub release](https://github.com/FrostWalk/pipeline-alert/releases) for the matching tag.

Install the package on the Raspberry Pi, edit `/etc/default/pipeline-horn-client`, then:

```bash
sudo systemctl enable --now pipeline-horn-client
sudo systemctl status pipeline-horn-client
```

The systemd unit passes `CLIENT_ARGS` to the client binary. Packaged default sound is `/usr/share/pipeline-horn-client/horn.mp3`. WAV files work with `aplay`; MP3 files require `mpg123`.

