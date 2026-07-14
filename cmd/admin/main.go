// admin is a command-line tool for managing an Open OSCAR Server instance via
// its Management API.
//
// The Management API base URL is read from the OSCAR_API environment variable
// (default http://127.0.0.1:8080).
//
// Usage:
//
//	admin <group> <subcommand> [flags]
//
// Run "admin help" for the list of commands.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/mk6i/open-oscar-server/client/admin"
)

const defaultBaseURL = "http://127.0.0.1:8080"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	group := os.Args[1]
	rest := os.Args[2:]

	switch group {
	case "version":
		runVersion(rest)
	case "users":
		runUsers(rest)
	case "sessions":
		runSessions(rest)
	case "rooms":
		runRooms(rest)
	case "directory":
		runDirectory(rest)
	case "keys":
		runKeys(rest)
	case "im":
		runIM(rest)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", group)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`admin - Open OSCAR Server Management CLI

Usage: admin <group> <subcommand> [flags]

The base URL is read from OSCAR_API (default ` + defaultBaseURL + `).

Commands:
  version                          Show server build information
  users list [-json]               List all users
  users create -name N -pass P     Create a user
  users delete NAME                Delete a user
  users passwd NAME -pass P        Reset a user's password
  users suspend NAME               Suspend a user
  users unsuspend NAME             Clear a user's suspension
  sessions list [-json]            List active sessions
  sessions kick NAME               Disconnect a user's sessions
  rooms list [-json]               List public chat rooms
  rooms create NAME                Create a public chat room
  rooms delete NAME [NAME...]      Delete public chat rooms
  directory categories [-json]     List keyword categories
  directory keywords CAT_ID [-json] List keywords in a category
  directory add-cat NAME           Create a keyword category
  directory del-cat CAT_ID         Delete a keyword category
  directory add-kw CAT_ID NAME     Create a keyword in a category
  directory del-kw KW_ID           Delete a keyword
  keys list [-json]                List Web API keys
  keys create -app N [opts]        Create a Web API key
  keys delete ID                   Delete a Web API key
  keys toggle ID -enable|-disable  Enable or disable a Web API key
  im send -from F -to T -text M    Send an instant message

Global flags may appear after the subcommand. Use -h with any subcommand for
specific help. Use -json on list commands for machine-readable output.`)
}

// newClient builds a client from the OSCAR_API environment variable.
func newClient() *admin.Client {
	baseURL := os.Getenv("OSCAR_API")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	c, err := admin.New(baseURL, admin.WithTimeout(30*time.Second))
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid OSCAR_API base URL: %v\n", err)
		os.Exit(1)
	}
	return c
}

// fatal prints err to stderr and exits non-zero.
func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

// printJSON pretty-prints v as JSON to stdout.
func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fatal(err)
	}
}

// boolFlag embeds flag.FlagSet to support a shared -json option on list
// commands. fsJSON returns a configured FlagSet with a json flag.
func newListFlagSet(name string, asJSON *bool) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	fs.BoolVar(asJSON, "json", false, "output results as JSON")
	return fs
}

// runVersion handles "admin version".
func runVersion(args []string) {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "output as JSON")
	_ = fs.Parse(args)

	c := newClient()
	v, err := c.GetVersion(context.Background())
	if err != nil {
		fatal(err)
	}
	if *asJSON {
		printJSON(v)
		return
	}
	fmt.Printf("version: %s\ncommit: %s\ndate:    %s\n", v.Version, v.Commit, v.Date)
}

// runUsers handles the "admin users" group.
func runUsers(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: admin users <list|create|delete|passwd|suspend|unsuspend>")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		usersList(rest)
	case "create":
		usersCreate(rest)
	case "delete":
		usersSimple(rest, "delete", func(c *admin.Client, name string) {
			if err := c.DeleteUser(context.Background(), name); err != nil {
				fatal(err)
			}
			fmt.Printf("user %q deleted\n", name)
		})
	case "passwd":
		usersPasswd(rest)
	case "suspend":
		usersSimple(rest, "suspend", func(c *admin.Client, name string) {
			if err := c.SetSuspend(context.Background(), name, "suspended"); err != nil {
				fatal(err)
			}
			fmt.Printf("user %q suspended\n", name)
		})
	case "unsuspend":
		usersSimple(rest, "unsuspend", func(c *admin.Client, name string) {
			if err := c.SetSuspend(context.Background(), name, ""); err != nil {
				fatal(err)
			}
			fmt.Printf("user %q unsuspended\n", name)
		})
	default:
		fmt.Fprintf(os.Stderr, "unknown users subcommand %q\n", sub)
		os.Exit(1)
	}
}

func usersList(args []string) {
	var asJSON bool
	fs := newListFlagSet("users list", &asJSON)
	_ = fs.Parse(args)

	c := newClient()
	users, err := c.ListUsers(context.Background())
	if err != nil {
		fatal(err)
	}
	if asJSON {
		printJSON(users)
		return
	}
	if len(users) == 0 {
		fmt.Println("no users")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SCREEN NAME\tICQ?\tSUSPENDED\tBOT")
	for _, u := range users {
		fmt.Fprintf(w, "%s\t%v\t%s\t%v\n", u.ScreenName, u.IsICQ, u.SuspendedStatus, u.IsBot)
	}
	_ = w.Flush()
}

func usersCreate(args []string) {
	fs := flag.NewFlagSet("users create", flag.ExitOnError)
	name := fs.String("name", "", "screen name (required)")
	pass := fs.String("pass", "", "password (required)")
	_ = fs.Parse(args)
	if *name == "" || *pass == "" {
		fmt.Fprintln(os.Stderr, "usage: admin users create -name NAME -pass PASS")
		os.Exit(1)
	}
	c := newClient()
	if err := c.CreateUser(context.Background(), *name, *pass); err != nil {
		fatal(err)
	}
	fmt.Printf("user %q created\n", *name)
}

func usersPasswd(args []string) {
	fs := flag.NewFlagSet("users passwd", flag.ExitOnError)
	pass := fs.String("pass", "", "new password (required)")
	_ = fs.Parse(args)
	if fs.NArg() == 0 || *pass == "" {
		fmt.Fprintln(os.Stderr, "usage: admin users passwd NAME -pass PASS")
		os.Exit(1)
	}
	name := fs.Arg(0)
	c := newClient()
	if err := c.ResetPassword(context.Background(), name, *pass); err != nil {
		fatal(err)
	}
	fmt.Printf("password reset for %q\n", name)
}

// usersSimple runs a subcommand that takes a single positional screen name.
func usersSimple(args []string, name string, fn func(*admin.Client, string)) {
	fs := flag.NewFlagSet("users "+name, flag.ExitOnError)
	_ = fs.Parse(args)
	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "usage: admin users %s NAME\n", name)
		os.Exit(1)
	}
	fn(newClient(), fs.Arg(0))
}

// runSessions handles the "admin sessions" group.
func runSessions(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: admin sessions <list|kick>")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		sessionsList(rest)
	case "kick":
		fs := flag.NewFlagSet("sessions kick", flag.ExitOnError)
		_ = fs.Parse(rest)
		if fs.NArg() == 0 {
			fmt.Fprintln(os.Stderr, "usage: admin sessions kick NAME")
			os.Exit(1)
		}
		name := fs.Arg(0)
		c := newClient()
		if err := c.KickSession(context.Background(), name); err != nil {
			fatal(err)
		}
		fmt.Printf("sessions kicked for %q\n", name)
	default:
		fmt.Fprintf(os.Stderr, "unknown sessions subcommand %q\n", sub)
		os.Exit(1)
	}
}

func sessionsList(args []string) {
	var asJSON bool
	fs := newListFlagSet("sessions list", &asJSON)
	_ = fs.Parse(args)

	c := newClient()
	list, err := c.ListSessions(context.Background())
	if err != nil {
		fatal(err)
	}
	if asJSON {
		printJSON(list)
		return
	}
	fmt.Printf("%d active session(s)\n", list.Count)
	for _, s := range list.Sessions {
		fmt.Printf("- %s (online %ds, %d instance(s), icq=%v)\n", s.ScreenName, s.OnlineSeconds, s.InstanceCount, s.IsICQ)
	}
}

// runRooms handles the "admin rooms" group.
func runRooms(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: admin rooms <list|create|delete>")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		roomsList(rest)
	case "create":
		fs := flag.NewFlagSet("rooms create", flag.ExitOnError)
		_ = fs.Parse(rest)
		if fs.NArg() == 0 {
			fmt.Fprintln(os.Stderr, "usage: admin rooms create NAME")
			os.Exit(1)
		}
		name := fs.Arg(0)
		c := newClient()
		if err := c.CreatePublicRoom(context.Background(), name); err != nil {
			fatal(err)
		}
		fmt.Printf("room %q created\n", name)
	case "delete":
		fs := flag.NewFlagSet("rooms delete", flag.ExitOnError)
		_ = fs.Parse(rest)
		if fs.NArg() == 0 {
			fmt.Fprintln(os.Stderr, "usage: admin rooms delete NAME [NAME...]")
			os.Exit(1)
		}
		names := fs.Args()
		c := newClient()
		if err := c.DeletePublicRooms(context.Background(), names); err != nil {
			fatal(err)
		}
		fmt.Printf("deleted room(s): %s\n", strings.Join(names, ", "))
	default:
		fmt.Fprintf(os.Stderr, "unknown rooms subcommand %q\n", sub)
		os.Exit(1)
	}
}

func roomsList(args []string) {
	var asJSON bool
	fs := newListFlagSet("rooms list", &asJSON)
	_ = fs.Parse(args)

	c := newClient()
	rooms, err := c.ListPublicRooms(context.Background())
	if err != nil {
		fatal(err)
	}
	if asJSON {
		printJSON(rooms)
		return
	}
	if len(rooms) == 0 {
		fmt.Println("no public rooms")
		return
	}
	for _, r := range rooms {
		fmt.Printf("- %s (%d participant(s))\n", r.Name, len(r.Participants))
	}
}

// runDirectory handles the "admin directory" group.
func runDirectory(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: admin directory <categories|keywords|add-cat|del-cat|add-kw|del-kw>")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "categories":
		directoryCategories(rest)
	case "keywords":
		directoryKeywords(rest)
	case "add-cat":
		fs := flag.NewFlagSet("directory add-cat", flag.ExitOnError)
		_ = fs.Parse(rest)
		if fs.NArg() == 0 {
			fmt.Fprintln(os.Stderr, "usage: admin directory add-cat NAME")
			os.Exit(1)
		}
		c := newClient()
		cat, err := c.CreateCategory(context.Background(), fs.Arg(0))
		if err != nil {
			fatal(err)
		}
		fmt.Printf("category created: id=%d name=%s\n", cat.ID, cat.Name)
	case "del-cat":
		id := directoryIDArg(rest, "del-cat", "CAT_ID")
		c := newClient()
		if err := c.DeleteCategory(context.Background(), id); err != nil {
			fatal(err)
		}
		fmt.Printf("category %d deleted\n", id)
	case "add-kw":
		fs := flag.NewFlagSet("directory add-kw", flag.ExitOnError)
		_ = fs.Parse(rest)
		if fs.NArg() < 2 {
			fmt.Fprintln(os.Stderr, "usage: admin directory add-kw CAT_ID NAME")
			os.Exit(1)
		}
		id := atoiOrFatal(fs.Arg(0), "CAT_ID")
		c := newClient()
		kw, err := c.CreateKeyword(context.Background(), id, fs.Arg(1))
		if err != nil {
			fatal(err)
		}
		fmt.Printf("keyword created: id=%d name=%s\n", kw.ID, kw.Name)
	case "del-kw":
		id := directoryIDArg(rest, "del-kw", "KW_ID")
		c := newClient()
		if err := c.DeleteKeyword(context.Background(), id); err != nil {
			fatal(err)
		}
		fmt.Printf("keyword %d deleted\n", id)
	default:
		fmt.Fprintf(os.Stderr, "unknown directory subcommand %q\n", sub)
		os.Exit(1)
	}
}

func directoryCategories(args []string) {
	var asJSON bool
	fs := newListFlagSet("directory categories", &asJSON)
	_ = fs.Parse(args)

	c := newClient()
	cats, err := c.ListCategories(context.Background())
	if err != nil {
		fatal(err)
	}
	if asJSON {
		printJSON(cats)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME")
	for _, cat := range cats {
		fmt.Fprintf(w, "%d\t%s\n", cat.ID, cat.Name)
	}
	_ = w.Flush()
}

func directoryKeywords(args []string) {
	var asJSON bool
	fs := newListFlagSet("directory keywords", &asJSON)
	_ = fs.Parse(args)
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: admin directory keywords CAT_ID")
		os.Exit(1)
	}
	catID := atoiOrFatal(fs.Arg(0), "CAT_ID")

	c := newClient()
	kws, err := c.ListKeywords(context.Background(), catID)
	if err != nil {
		fatal(err)
	}
	if asJSON {
		printJSON(kws)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME")
	for _, kw := range kws {
		fmt.Fprintf(w, "%d\t%s\n", kw.ID, kw.Name)
	}
	_ = w.Flush()
}

// directoryIDArg parses a single integer ID positional argument.
func directoryIDArg(args []string, cmd, label string) int {
	fs := flag.NewFlagSet("directory "+cmd, flag.ExitOnError)
	_ = fs.Parse(args)
	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "usage: admin directory %s %s\n", cmd, label)
		os.Exit(1)
	}
	return atoiOrFatal(fs.Arg(0), label)
}

// atoiOrFatal parses s as an int or exits.
func atoiOrFatal(s, label string) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		fmt.Fprintf(os.Stderr, "invalid %s %q: %v\n", label, s, err)
		os.Exit(1)
	}
	return n
}

// runKeys handles the "admin keys" group.
func runKeys(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: admin keys <list|create|delete|toggle>")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		keysList(rest)
	case "create":
		keysCreate(rest)
	case "delete":
		fs := flag.NewFlagSet("keys delete", flag.ExitOnError)
		_ = fs.Parse(rest)
		if fs.NArg() == 0 {
			fmt.Fprintln(os.Stderr, "usage: admin keys delete ID")
			os.Exit(1)
		}
		id := fs.Arg(0)
		c := newClient()
		if err := c.DeleteWebAPIKey(context.Background(), id); err != nil {
			fatal(err)
		}
		fmt.Printf("key %s deleted\n", id)
	case "toggle":
		keysToggle(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown keys subcommand %q\n", sub)
		os.Exit(1)
	}
}

func keysList(args []string) {
	var asJSON bool
	fs := newListFlagSet("keys list", &asJSON)
	_ = fs.Parse(args)

	c := newClient()
	keys, err := c.ListWebAPIKeys(context.Background())
	if err != nil {
		fatal(err)
	}
	if asJSON {
		printJSON(keys)
		return
	}
	if len(keys) == 0 {
		fmt.Println("no keys")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DEV ID\tAPP NAME\tACTIVE\tRATE LIMIT")
	for _, k := range keys {
		fmt.Fprintf(w, "%s\t%s\t%v\t%d/min\n", k.DevID, k.AppName, k.IsActive, k.RateLimit)
	}
	_ = w.Flush()
}

func keysCreate(args []string) {
	fs := flag.NewFlagSet("keys create", flag.ExitOnError)
	app := fs.String("app", "", "application name (required)")
	originsCSV := fs.String("origins", "", "comma-separated allowed CORS origins")
	rate := fs.Int("rate", 0, "requests per minute (server default 60 when omitted)")
	capsCSV := fs.String("capabilities", "", "comma-separated capabilities")
	_ = fs.Parse(args)
	if *app == "" {
		fmt.Fprintln(os.Stderr, "usage: admin keys create -app NAME [-origins CSV] [-rate N] [-capabilities CSV]")
		os.Exit(1)
	}

	req := admin.CreateWebAPIKeyRequest{AppName: *app, RateLimit: *rate}
	if *originsCSV != "" {
		req.AllowedOrigins = splitCSV(*originsCSV)
	}
	if *capsCSV != "" {
		req.Capabilities = splitCSV(*capsCSV)
	}

	c := newClient()
	key, err := c.CreateWebAPIKey(context.Background(), req)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("key created\n")
	fmt.Printf("  dev id:  %s\n", key.DevID)
	fmt.Printf("  dev key: %s\n", key.DevKey)
	fmt.Printf("  app:     %s\n", key.AppName)
	fmt.Println("  NOTE: the dev key is only shown once.")
}

func keysToggle(args []string) {
	fs := flag.NewFlagSet("keys toggle", flag.ExitOnError)
	enable := fs.Bool("enable", false, "enable the key")
	disable := fs.Bool("disable", false, "disable the key")
	_ = fs.Parse(args)
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: admin keys toggle ID -enable|-disable")
		os.Exit(1)
	}
	id := fs.Arg(0)
	if !*enable && !*disable {
		fmt.Fprintln(os.Stderr, "error: must pass -enable or -disable")
		os.Exit(1)
	}
	c := newClient()
	if _, err := c.ToggleWebAPIKey(context.Background(), id, *enable); err != nil {
		fatal(err)
	}
	state := "enabled"
	if *disable {
		state = "disabled"
	}
	fmt.Printf("key %s %s\n", id, state)
}

// runIM handles the "admin im" group.
func runIM(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: admin im send -from F -to T -text M")
		os.Exit(1)
	}
	sub := args[0]
	if sub != "send" {
		fmt.Fprintf(os.Stderr, "unknown im subcommand %q\n", sub)
		os.Exit(1)
	}
	rest := args[1:]
	fs := flag.NewFlagSet("im send", flag.ExitOnError)
	from := fs.String("from", "", "sender screen name")
	to := fs.String("to", "", "recipient screen name")
	text := fs.String("text", "", "message text")
	_ = fs.Parse(rest)
	if *from == "" || *to == "" || *text == "" {
		fmt.Fprintln(os.Stderr, "usage: admin im send -from FROM -to TO -text TEXT")
		os.Exit(1)
	}
	c := newClient()
	if err := c.SendIM(context.Background(), *from, *to, *text); err != nil {
		fatal(err)
	}
	fmt.Printf("message sent from %q to %q\n", *from, *to)
}

// splitCSV trims and splits a comma-separated string into fields.
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
