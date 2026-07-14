# Clients: Libraries and the `oscar-admin` CLI

Open OSCAR Server ships two reusable Go libraries for building tools that talk to
the server, plus a command-line management tool. This guide covers the libraries'
APIs and the CLI's commands. For runnable bots and protocol bridges built on top
of these libraries, see [BRIDGES.md](BRIDGES.md). For connecting classic AIM
clients, see [CLIENT.md](CLIENT.md).

| Library         | Import path                                            | Talks to                |
|-----------------|--------------------------------------------------------|-------------------------|
| `client/toc`    | `github.com/mk6i/open-oscar-server/client/toc`        | TOC server (port 9898)  |
| `client/admin`  | `github.com/mk6i/open-oscar-server/client/admin`      | Management API (8080)   |

Both depend only on the Go standard library.

## `client/toc` — TOC protocol client

A client for the text-based TOC protocol spoken by the Open OSCAR Server TOC
endpoint. It handles SFLAP/FLAP framing, the `FLAPON` sign-on handshake, TOC
password roasting, and the common commands a bot needs.

### Types and methods

```go
// Connect to a TOC server. addr is host:port.
func Dial(addr string, opts Options) (*Client, error)

// Options configures a Client. The zero value is usable but receives no events.
type Options struct {
    Handler     Handler        // typed callbacks for common message types (optional)
    OnEvent     EventHandler   // receives every parsed message, before Handler (optional)
    KeepAlive   time.Duration  // keep-alive FLAP interval during Receive; 0 disables
    DialTimeout time.Duration  // TCP dial timeout; 0 means no timeout
}

// A connected TOC client, safe for concurrent use.
type Client struct { ... }

func (c *Client) SignIn(screenName, password string) error   // sign-on handshake + toc_init_done
func (c *Client) SendIM(to, text string) error               // toc_send_im
func (c *Client) SetAway(msg string) error                   // "" clears away / marks available
func (c *Client) SetInfo(info string) error                  // toc_set_info (profile HTML)
func (c *Client) AddBuddy(names ...string) error             // toc_add_buddy
func (c *Client) SendCommand(cmd string) error               // send an arbitrary TOC command
func (c *Client) Receive(ctx context.Context) error          // run the receive loop (blocks)
func (c *Client) Close() error                               // best-effort signoff + close
func (c *Client) ScreenName() string                         // name from SignIn, or "" before sign-in

// SignInError is returned by SignIn when the server replies ERROR:<code>.
type SignInError struct{ Code string }
```

Handlers receive decoded server messages from the `Receive` loop. They are
invoked on the `Receive` goroutine and must not block on the connection.

```go
type Handler interface {
    OnIM(from, text string, autoResponse bool) // autoResponse is true for away replies
    OnError(code string)                        // text after "ERROR:"
}

// OnEvent receives every parsed Event and is invoked before the typed Handler.
type EventHandler func(c *Client, ev Event)
```

`Event` exposes `Type` (`EventIM`, `EventError`, `EventSignOn`, `EventConfig`,
`EventNick`, `EventUpdateBuddy`, or `EventOther`), the full `Raw` message, and
the fields relevant to the type (`From`, `Text`, `Auto`, `Code`, `Name`).

### Example: connect, sign in, send and receive an IM

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/mk6i/open-oscar-server/client/toc"
)

type echoHandler struct {
	client *toc.Client
}

func (h *echoHandler) OnIM(from, text string, auto bool) {
	if auto {
		return
	}
	log.Printf("IM from %s: %s", from, text)
	if err := h.client.SendIM(from, "echo: "+text); err != nil {
		log.Printf("reply failed: %v", err)
	}
}

func (h *echoHandler) OnError(code string) {
	log.Printf("server error: ERROR:%s", code)
}

func main() {
	h := &echoHandler{}
	c, err := toc.Dial("127.0.0.1:9898", toc.Options{
		Handler:     h,
		KeepAlive:   60 * time.Second,
		DialTimeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	h.client = c

	if err := c.SignIn("echobot", "secret"); err != nil {
		log.Fatal(err)
	}
	log.Printf("signed in as %s", c.ScreenName())

	if err := c.SendIM("buddy", "hello there"); err != nil {
		log.Fatal(err)
	}

	log.Fatal(c.Receive(context.Background()))
}
```

## `client/admin` — Management API client

A client for the HTTP Management API that runs on port 8080 by default (see
`api.yml` for the full endpoint reference). It is safe for concurrent use.

### Creating a client

```go
func New(baseURL string, opts ...Option) (*Client, error) // baseURL must include scheme + host

func WithHTTPClient(h *http.Client) Option // override the underlying *http.Client
func WithTimeout(d time.Duration) Option    // set a per-request timeout

// ErrorResponse is returned for any non-2xx response.
type ErrorResponse struct {
	StatusCode int
	Body       string
}
```

### Example: list users and send an IM

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mk6i/open-oscar-server/client/admin"
)

func main() {
	c, err := admin.New("http://127.0.0.1:8080", admin.WithTimeout(30*time.Second))
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	users, err := c.ListUsers(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, u := range users {
		fmt.Printf("- %s (icq=%v, suspended=%q, bot=%v)\n",
			u.ScreenName, u.IsICQ, u.SuspendedStatus, u.IsBot)
	}

	if err := c.SendIM(ctx, "admin", "buddy", "hello from the API"); err != nil {
		log.Fatal(err)
	}
}
```

### Method reference

All methods take a `context.Context` as their first argument. The HTTP method
and path each method maps to are noted for reference.

#### Server

| Method                                        | HTTP            | Description                              |
|-----------------------------------------------|-----------------|------------------------------------------|
| `GetVersion(ctx) (Version, error)`            | `GET /version`  | Build version, commit, and date.         |

#### Users and accounts

| Method                                                                 | HTTP                                  | Description                                                        |
|------------------------------------------------------------------------|---------------------------------------|--------------------------------------------------------------------|
| `ListUsers(ctx) ([]User, error)`                                       | `GET /user`                           | All registered accounts.                                           |
| `CreateUser(ctx, screenName, password string) error`                   | `POST /user`                          | Create an AIM or ICQ account.                                      |
| `DeleteUser(ctx, screenName string) error`                             | `DELETE /user`                        | Delete an account.                                                 |
| `ResetPassword(ctx, screenName, password string) error`                | `PUT /user/password`                  | Set a new password.                                                |
| `GetAccount(ctx, screenName string) (Account, error)`                  | `GET /user/{sn}/account`              | Full account details (profile, email, reg status, etc.).           |
| `PatchAccount(ctx, screenName, req PatchAccountRequest) error`         | `PATCH /user/{sn}/account`            | Partial update (`SuspendedStatus`, `IsBot`).                       |
| `SetSuspend(ctx, screenName, status string) error`                     | *(patch)*                             | Set/clear suspension (`""`, `deleted`, `expired`, `suspended`, `suspended_age`). |
| `ToggleBot(ctx, screenName string, isBot bool) error`                  | *(patch)*                             | Set the bot flag.                                                  |
| `ListLinkedAccounts(ctx, screenName string) (LinkedAccounts, error)`   | `GET /user/{sn}/linked-account`       | Screen names linked to the account.                                |
| `AddLinkedAccount(ctx, screenName, linkedScreenName string) error`     | `POST /user/{sn}/linked-account`      | Link a screen name.                                                |
| `RemoveLinkedAccount(ctx, screenName, linkedScreenName string) error`  | `DELETE /user/{sn}/linked-account/{l}`| Remove a link.                                                     |

#### Sessions

| Method                                                        | HTTP                 | Description                                  |
|---------------------------------------------------------------|----------------------|----------------------------------------------|
| `ListSessions(ctx) (SessionList, error)`                      | `GET /session`       | All active sessions.                         |
| `GetSessions(ctx, screenName string) (SessionList, error)`    | `GET /session/{sn}`  | Active sessions for one user.                |
| `KickSession(ctx, screenName string) error`                   | `DELETE /session/{sn}` | Disconnect a user's sessions.              |

#### Chat rooms

| Method                                                           | HTTP                       | Description                          |
|------------------------------------------------------------------|----------------------------|--------------------------------------|
| `ListPublicRooms(ctx) ([]ChatRoom, error)`                       | `GET /chat/room/public`    | Public AIM chat rooms.               |
| `ListPrivateRooms(ctx) ([]ChatRoom, error)`                      | `GET /chat/room/private`   | Private AIM chat rooms.              |
| `CreatePublicRoom(ctx, name string) error`                       | `POST /chat/room/public`   | Create a public chat room.           |
| `DeletePublicRooms(ctx, names []string) error`                   | `DELETE /chat/room/public` | Delete one or more public rooms.     |

#### Feedbag (buddy lists)

| Method                                                                                                    | HTTP                                                                     | Description                          |
|-----------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------|--------------------------------------|
| `GetFeedbag(ctx, screenName string) ([]FeedbagGroup, error)`                                              | `GET /feedbag/{sn}/group`                                                | Buddy list grouped by group ID.      |
| `CreateFeedbagGroup(ctx, screenName, groupName string) (FeedbagGroupResult, error)`                       | `PUT /feedbag/{sn}/group/{group_name}`                                   | Create a group (idempotent).         |
| `AddBuddy(ctx, screenName string, groupID int, buddyScreenName string) (FeedbagBuddyResult, error)`       | `PUT /feedbag/{sn}/group/{group_id}/buddy/{buddy}`                       | Add a buddy (idempotent).            |
| `RemoveBuddy(ctx, screenName string, groupID int, buddyScreenName string) error`                          | `DELETE /feedbag/{sn}/group/{group_id}/buddy/{buddy}`                    | Remove a buddy.                      |

#### Directory (keyword categories)

| Method                                                                   | HTTP                                | Description                          |
|--------------------------------------------------------------------------|-------------------------------------|--------------------------------------|
| `ListCategories(ctx) ([]DirectoryCategory, error)`                       | `GET /directory/category`           | All keyword categories.              |
| `CreateCategory(ctx, name string) (DirectoryCategory, error)`            | `POST /directory/category`          | Create a category.                   |
| `DeleteCategory(ctx, categoryID int) error`                              | `DELETE /directory/category/{id}`   | Delete a category.                   |
| `ListKeywords(ctx, categoryID int) ([]DirectoryKeyword, error)`          | `GET /directory/category/{id}/keyword` | Keywords in a category.           |
| `CreateKeyword(ctx, categoryID int, name string) (DirectoryKeyword, error)` | `POST /directory/keyword`        | Create a keyword in a category.      |
| `DeleteKeyword(ctx, keywordID int) error`                                | `DELETE /directory/keyword/{id}`    | Delete a keyword.                    |

#### Instant messages

| Method                                                        | HTTP                       | Description                                                                              |
|---------------------------------------------------------------|----------------------------|------------------------------------------------------------------------------------------|
| `SendIM(ctx, from, to, text string) error`                    | `POST /instant-message`    | Send an IM. `from` need not exist; no error if `to` is offline or unknown.               |

#### Web API keys

| Method                                                                                | HTTP                          | Description                                                       |
|---------------------------------------------------------------------------------------|-------------------------------|-------------------------------------------------------------------|
| `ListWebAPIKeys(ctx) ([]WebAPIKey, error)`                                            | `GET /admin/webapi/keys`      | All Web API keys (`DevKey` never returned here).                  |
| `GetWebAPIKey(ctx, devID string) (WebAPIKey, error)`                                  | `GET /admin/webapi/keys/{id}` | One key by developer ID.                                          |
| `CreateWebAPIKey(ctx, req CreateWebAPIKeyRequest) (WebAPIKey, error)`                 | `POST /admin/webapi/keys`     | Create a key. `DevKey` is returned only at creation time.         |
| `UpdateWebAPIKey(ctx, devID string, req UpdateWebAPIKeyRequest) (WebAPIKey, error)`   | `PUT /admin/webapi/keys/{id}` | Partial update (only non-nil fields applied).                     |
| `DeleteWebAPIKey(ctx, devID string) error`                                            | `DELETE /admin/webapi/keys/{id}` | Delete a key permanently.                                      |
| `ToggleWebAPIKey(ctx, devID string, active bool) (WebAPIKey, error)`                  | *(update)*                    | Enable or disable a key.                                          |

## The `oscar-admin` CLI

`oscar-admin` is a command-line wrapper around `client/admin`. The base URL of
the Management API is read from the `OSCAR_API` environment variable and defaults
to `http://127.0.0.1:8080` (a 30-second request timeout is applied).

### Install

```shell
make admin
```

This builds the binary as `./oscar-admin` from `cmd/admin`.

### Configuration

| Variable    | Required | Default                  | Description                              |
|-------------|----------|--------------------------|------------------------------------------|
| `OSCAR_API` | No       | `http://127.0.0.1:8080`  | Base URL of the Management API.          |

### Commands

Every command group is invoked as `oscar-admin <group> <subcommand> [flags]`.
Run `oscar-admin help` for the built-in summary; pass `-h` to any subcommand for
its flags. List-style subcommands accept `-json` for machine-readable output.

#### `version`

Show server build information.

```shell
oscar-admin version
oscar-admin version -json
```

#### `users`

```shell
oscar-admin users list [-json]                 # list all users
oscar-admin users create -name NAME -pass PASS # create a user
oscar-admin users delete NAME                  # delete a user
oscar-admin users passwd NAME -pass PASS       # reset a user's password
oscar-admin users suspend NAME                 # suspend a user
oscar-admin users unsuspend NAME               # clear a user's suspension
```

| Subcommand  | Flags / args                              |
|-------------|-------------------------------------------|
| `list`      | `-json`                                   |
| `create`    | `-name` (required), `-pass` (required)    |
| `delete`    | `NAME` (positional, required)             |
| `passwd`    | `NAME` (positional, required), `-pass` (required) |
| `suspend`   | `NAME` (positional, required)             |
| `unsuspend` | `NAME` (positional, required)             |

#### `sessions`

```shell
oscar-admin sessions list [-json]   # list active sessions
oscar-admin sessions kick NAME      # disconnect a user's sessions
```

| Subcommand | Flags / args                      |
|------------|-----------------------------------|
| `list`     | `-json`                           |
| `kick`     | `NAME` (positional, required)     |

#### `rooms`

```shell
oscar-admin rooms list [-json]          # list public chat rooms
oscar-admin rooms create NAME           # create a public chat room
oscar-admin rooms delete NAME [NAME...] # delete one or more public rooms
```

| Subcommand | Flags / args                                  |
|------------|-----------------------------------------------|
| `list`     | `-json`                                       |
| `create`   | `NAME` (positional, required)                 |
| `delete`   | `NAME [NAME...]` (one or more positionals)    |

#### `directory`

Manage keyword categories and keywords.

```shell
oscar-admin directory categories [-json]        # list keyword categories
oscar-admin directory keywords CAT_ID [-json]   # list keywords in a category
oscar-admin directory add-cat NAME              # create a keyword category
oscar-admin directory del-cat CAT_ID            # delete a keyword category
oscar-admin directory add-kw CAT_ID NAME        # create a keyword in a category
oscar-admin directory del-kw KW_ID              # delete a keyword
```

| Subcommand    | Flags / args                                            |
|---------------|---------------------------------------------------------|
| `categories`  | `-json`                                                 |
| `keywords`    | `CAT_ID` (positional, required), `-json`                |
| `add-cat`     | `NAME` (positional, required)                           |
| `del-cat`     | `CAT_ID` (positional integer, required)                 |
| `add-kw`      | `CAT_ID NAME` (two positionals, required)               |
| `del-kw`      | `KW_ID` (positional integer, required)                  |

#### `keys`

Manage Web API keys (for the Web AIM-style API). The dev key is shown only once
at creation time.

```shell
oscar-admin keys list [-json]                                  # list Web API keys
oscar-admin keys create -app NAME [-origins CSV] [-rate N] [-capabilities CSV]
oscar-admin keys delete ID                                    # delete a key
oscar-admin keys toggle ID -enable                            # enable a key
oscar-admin keys toggle ID -disable                           # disable a key
```

| Subcommand | Flags / args                                                                       |
|------------|------------------------------------------------------------------------------------|
| `list`     | `-json`                                                                            |
| `create`   | `-app` (required), `-origins` (CSV), `-rate` (int), `-capabilities` (CSV)          |
| `delete`   | `ID` (positional, required)                                                        |
| `toggle`   | `ID` (positional, required), plus exactly one of `-enable` or `-disable`           |

JSON output example:

```shell
OSCAR_API=http://127.0.0.1:8080 oscar-admin users list -json
```

#### `im`

Send an instant message.

```shell
oscar-admin im send -from FROM -to TO -text TEXT
```

| Subcommand | Flags                                                       |
|------------|-------------------------------------------------------------|
| `send`     | `-from` (required), `-to` (required), `-text` (required)    |

The sender screen name does not need to exist; no error is raised if the
recipient is offline or unknown.
