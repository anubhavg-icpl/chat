# Bots and Bridges

Open OSCAR Server ships several runnable bots and protocol bridges built on the
[`client/toc`](CLIENTS.md#clienttoc--toc-protocol-client) library. This guide
covers what each one does, its configuration, and how to run it. For the
underlying client libraries and the `oscar-admin` CLI, see
[CLIENTS.md](CLIENTS.md). For running the server itself, see
[BUILD.md](BUILD.md) or [DOCKER.md](DOCKER.md).

| Binary           | Built from            | What it does                                            |
|------------------|-----------------------|---------------------------------------------------------|
| `tocbot`         | `cmd/tocbot`          | Simple command-driven TOC bot.                          |
| `aibot`          | `cmd/aibot`           | LLM-powered AIM bot (OpenAI-compatible providers).      |
| `discord-bridge` | `cmd/discord-bridge`  | Relays a Discord channel to and from an AIM user.       |
| `irc-bridge`     | `cmd/irc-bridge`      | Relays an IRC channel to and from an AIM user.          |

Each bot/bridge signs in to the TOC server as an AIM account, so create the
account first (for example with `oscar-admin users create -name <bot> -pass <pw>`
or the `client/admin` library). All of them honor `TOCBOT_SERVER`,
`TOCBOT_SCREENNAME`, and `TOCBOT_PASSWORD` for the AIM side of the connection.

## `tocbot`

A small, self-contained TOC bot. It connects, signs in, and replies to a few
instant-message commands. It reconnects automatically with exponential backoff
(starting at 2 seconds, doubling up to a 60-second cap) after a disconnect.

### Commands

Messages are matched exactly (case-sensitive) except for `!echo`, which takes an
argument.

| Command         | Reply                                                            |
|-----------------|------------------------------------------------------------------|
| `!help`         | Lists the available commands.                                    |
| `!echo <text>`  | Replies with `<text>`. `!echo` alone replies with an empty IM.   |
| `!time`         | Replies with the current time (`time.RFC1123`).                  |
| `!ping`         | Replies `pong`.                                                  |
| *(anything else)* | Echoes the received text back.                                 |

Automatic (away) replies are ignored.

### Environment

| Variable           | Required | Default            | Description                                          |
|--------------------|----------|--------------------|------------------------------------------------------|
| `TOCBOT_SERVER`    | No       | `127.0.0.1:9898`   | TOC server address (`host:port`).                    |
| `TOCBOT_SCREENNAME`| Yes      |                    | AIM screen name to sign in with.                     |
| `TOCBOT_PASSWORD`  | Yes      |                    | Password for the screen name.                        |
| `TOCBOT_AWAY`      | No       | *(online)*         | Away message. Empty means online.                    |

### Run

```shell
make tocbot
TOCBOT_SCREENNAME=tocbot TOCBOT_PASSWORD=secret ./tocbot
```

## `aibot`

An AIM bot that forwards incoming instant messages to a large language model and
replies with the generated response. It speaks the OpenAI Chat Completions API
(`POST {base_url}/chat/completions`), so it works with any compatible provider.

### How it works

1. Each incoming IM is appended to that sender's per-user conversation history.
2. The bot builds a request of `[system prompt] + [history]` and calls the
   completer with a 60-second timeout.
3. The assistant reply is appended to the history and sent back to the user.
4. If the model call fails, a short apology message is sent instead.

History is capped per user at `AIBOT_HISTORY_LIMIT` messages (oldest dropped).

### Provider-agnostic

Because it targets the standard OpenAI Chat Completions endpoint, `aibot` works
with any OpenAI-compatible backend by setting `OPENAI_BASE_URL` and
`OPENAI_MODEL`. Common options:

- **OpenAI**: `OPENAI_BASE_URL=https://api.openai.com/v1` (default),
  `OPENAI_MODEL=gpt-4o-mini` (default).
- **OpenRouter**: `OPENAI_BASE_URL=https://openrouter.ai/api/v1`,
  `OPENAI_MODEL=<model-id>`.
- **Ollama**: `OPENAI_BASE_URL=http://localhost:11434/v1`,
  `OPENAI_MODEL=<model>`.
- **LM Studio**: `OPENAI_BASE_URL=http://localhost:1234/v1`,
  `OPENAI_MODEL=<model>`.

For local providers that do not require a key, `OPENAI_API_KEY` must still be set
to a non-empty value (the bot validates it is present) — any placeholder string
works.

### Environment

| Variable                 | Required | Default                          | Description                                                       |
|--------------------------|----------|----------------------------------|-------------------------------------------------------------------|
| `OPENAI_BASE_URL`        | No       | `https://api.openai.com/v1`      | OpenAI-compatible API base URL.                                   |
| `OPENAI_API_KEY`         | Yes      |                                  | API key (bearer token). Must be non-empty.                        |
| `OPENAI_MODEL`           | No       | `gpt-4o-mini`                    | Model id passed in the request.                                   |
| `AIBOT_SYSTEM_PROMPT`    | No       | *(built-in friendly prompt)*     | System prompt prepended to each conversation.                     |
| `AIBOT_HISTORY_LIMIT`    | No       | `8`                              | Max messages kept per user (oldest dropped). Non-integers fall back to the default.|
| `TOCBOT_SERVER`          | No       | `127.0.0.1:9898`                 | TOC server address (`host:port`).                                 |
| `TOCBOT_SCREENNAME`      | Yes      |                                  | AIM screen name to sign in with.                                  |
| `TOCBOT_PASSWORD`        | Yes      |                                  | Password for the screen name.                                     |

### Run

```shell
make aibot
OPENAI_API_KEY=sk-... \
TOCBOT_SCREENNAME=aibot TOCBOT_PASSWORD=secret \
./aibot
```

Example with a local Ollama model:

```shell
make aibot
OPENAI_BASE_URL=http://localhost:11434/v1 \
OPENAI_API_KEY=ollama \
OPENAI_MODEL=llama3.1 \
TOCBOT_SCREENNAME=aibot TOCBOT_PASSWORD=secret \
./aibot
```

## `discord-bridge`

Relays messages between a single Discord channel and a single AIM user via TOC.
The bot listens for messages in the watched channel and delivers them to the AIM
user, and posts AIM instant messages it receives into the channel.

### Relay direction

- **Discord to AIM**: messages in the watched channel are sent to the AIM user
  named `AIM_TO`. The bridge's own posts are ignored to avoid loops. When
  `BRIDGE_TRIGGER` is set, only messages starting with that prefix are relayed
  and the prefix is stripped. Each message is formatted as
  `<author>: <content>` and truncated to 2000 runes.
- **AIM to Discord**: non-automatic instant messages received by the bridge's
  AIM account are posted into the watched channel as
  `<from>: <text>` (truncated to 2000 runes).

### Setup

1. Create a Discord application and add a bot at the
   [Discord Developer Portal](https://discord.com/developers/applications).
2. Under the bot's settings, enable the **Message Content Intent**.
3. Copy the **bot token** and set it as `DISCORD_TOKEN`.
4. Invite the bot to your server.
5. Enable Developer Mode in Discord, right-click the channel to bridge, and copy
   its ID into `DISCORD_CHANNEL_ID`.
6. Create an AIM account for the bridge and set `TOCBOT_SCREENNAME` /
   `TOCBOT_PASSWORD`, and set `AIM_TO` to the AIM user it should talk to.

### Environment

| Variable             | Required | Default            | Description                                                              |
|----------------------|----------|--------------------|--------------------------------------------------------------------------|
| `DISCORD_TOKEN`      | Yes      |                    | Discord bot token.                                                       |
| `DISCORD_CHANNEL_ID` | Yes      |                    | ID of the Discord channel to bridge.                                     |
| `AIM_TO`             | Yes      |                    | AIM screen name receiving Discord messages (and sending to the channel).|
| `TOCBOT_SERVER`      | No       | `127.0.0.1:9898`   | TOC server address (`host:port`).                                        |
| `TOCBOT_SCREENNAME`  | Yes      |                    | AIM screen name the bridge signs in as.                                  |
| `TOCBOT_PASSWORD`    | Yes      |                    | Password for the AIM account.                                            |
| `BRIDGE_TRIGGER`     | No       | *(empty: all)*     | Optional prefix a Discord message must start with to be relayed.         |

### Run

```shell
make discord-bridge
DISCORD_TOKEN=... DISCORD_CHANNEL_ID=1234567890 \
AIM_TO=buddy TOCBOT_SCREENNAME=aimbridge TOCBOT_PASSWORD=secret \
./discord-bridge
```

## `irc-bridge`

Relays messages between a single IRC channel and a single AIM user via TOC. The
bridge joins the IRC channel, and relays channel traffic in both directions.

### Relay direction

- **IRC to AIM**: channel `PRIVMSG`s (except the bridge's own) are sent to the
  AIM user named `AIM_TO`, formatted as `<nick>: <text>`.
- **AIM to IRC**: instant messages received by the bridge's AIM account are sent
  to the IRC channel as `PRIVMSG`s, formatted as `<from>: <text>`. All IMs are
  relayed, including automatic (away) replies.

The bridge answers IRC `PING`s and shuts down gracefully (sends `QUIT`, closes
TOC) on `SIGINT` / `SIGTERM`. Set `IRC_TLS=true` to connect over TLS.

### Environment

| Variable           | Required | Default            | Description                                          |
|--------------------|----------|--------------------|------------------------------------------------------|
| `IRC_SERVER`       | Yes      |                    | IRC server hostname.                                 |
| `IRC_PORT`         | No       | `6667`             | IRC server port.                                     |
| `IRC_TLS`          | No       | `false`            | Connect over TLS when `true`.                        |
| `IRC_NICK`         | Yes      |                    | IRC nickname the bridge uses.                        |
| `IRC_CHANNEL`      | Yes      |                    | IRC channel to join and bridge.                      |
| `AIM_TO`           | Yes      |                    | AIM screen name that receives IRC messages.          |
| `TOCBOT_SERVER`    | No       | `127.0.0.1:9898`   | TOC server address (`host:port`).                    |
| `TOCBOT_SCREENNAME`| Yes      |                    | AIM screen name the bridge signs in as.              |
| `TOCBOT_PASSWORD`  | Yes      |                    | Password for the AIM account.                        |

### Run

```shell
make irc-bridge
IRC_SERVER=irc.example.org IRC_NICK=aimbridge IRC_CHANNEL='#aim' \
AIM_TO=buddy TOCBOT_SCREENNAME=aimbridge TOCBOT_PASSWORD=secret \
./irc-bridge
```

To use TLS (for example on port 6697):

```shell
make irc-bridge
IRC_SERVER=irc.example.org IRC_PORT=6697 IRC_TLS=true IRC_NICK=aimbridge IRC_CHANNEL='#aim' \
AIM_TO=buddy TOCBOT_SCREENNAME=aimbridge TOCBOT_PASSWORD=secret \
./irc-bridge
```

## Build targets

All bots and bridges are built with `go build` via these Make targets:

| Target             | Builds            | Output binary      |
|--------------------|-------------------|--------------------|
| `make tocbot`      | `cmd/tocbot`      | `./tocbot`         |
| `make aibot`       | `cmd/aibot`       | `./aibot`          |
| `make discord-bridge` | `cmd/discord-bridge` | `./discord-bridge` |
| `make irc-bridge`  | `cmd/irc-bridge`  | `./irc-bridge`     |
| `make clients`     | tocbot + admin    | `./tocbot`, `./oscar-admin` |
| `make bridges`     | discord + irc bridges | `./discord-bridge`, `./irc-bridge` |

Containerized: see docker-compose.yaml `bots` profile — `docker compose --profile bots up -d --build`.
