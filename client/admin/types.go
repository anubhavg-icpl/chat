package admin

import "time"

// Version describes build information for the running server.
type Version struct {
	// Version is the release version number.
	Version string `json:"version"`
	// Commit is the git commit hash of the build.
	Commit string `json:"commit"`
	// Date is the build date and timestamp in RFC3339 format.
	Date string `json:"date"`
}

// User describes a registered user account, as returned by GET /user.
type User struct {
	// ID is the user's unique identifier.
	ID string `json:"id"`
	// ScreenName is the user's AIM screen name or ICQ UIN.
	ScreenName string `json:"screen_name"`
	// IsICQ is true when the account is an ICQ user instead of an AIM user.
	IsICQ bool `json:"is_icq"`
	// SuspendedStatus is the textual suspended status of the account.
	SuspendedStatus string `json:"suspended_status"`
	// IsBot indicates whether the account is a bot.
	IsBot bool `json:"is_bot"`
}

// Account describes the full account details for a single user, as returned by
// GET /user/{screenname}/account.
type Account struct {
	// ID is the user's unique identifier.
	ID string `json:"id"`
	// ScreenName is the user's AIM screen name or ICQ UIN.
	ScreenName string `json:"screen_name"`
	// Profile is the user's AIM profile HTML.
	Profile string `json:"profile"`
	// EmailAddress is the user's email address.
	EmailAddress string `json:"email_address"`
	// RegStatus is the user's registration disclosure status.
	RegStatus uint16 `json:"reg_status"`
	// Confirmed is the user's account confirmation status.
	Confirmed bool `json:"confirmed"`
	// IsICQ is true when the account is an ICQ user instead of an AIM user.
	IsICQ bool `json:"is_icq"`
	// SuspendedStatus is the textual suspended status of the account.
	SuspendedStatus string `json:"suspended_status"`
	// IsBot indicates whether the account is a bot.
	IsBot bool `json:"is_bot"`
}

// SessionInstance describes a single concurrent client connection for a
// session.
type SessionInstance struct {
	// Num is the instance number for this session instance.
	Num int `json:"num"`
	// IdleSeconds is the number of seconds this instance has been idle.
	IdleSeconds int `json:"idle_seconds"`
	// IsAway is true when this instance is away.
	IsAway bool `json:"is_away"`
	// AwayMessage is this instance's AIM away message HTML.
	AwayMessage string `json:"away_message"`
	// IsInvisible is true when this instance is invisible.
	IsInvisible bool `json:"is_invisible"`
	// RemoteAddr is the remote IP address of the connection.
	RemoteAddr string `json:"remote_addr"`
	// RemotePort is the remote port number of the connection.
	RemotePort int `json:"remote_port"`
}

// Session describes an active user session.
type Session struct {
	// ID is the user's unique identifier.
	ID string `json:"id"`
	// ScreenName is the user's AIM screen name or ICQ UIN.
	ScreenName string `json:"screen_name"`
	// OnlineSeconds is the number of seconds the session has been online.
	OnlineSeconds int `json:"online_seconds"`
	// IsAway is true when the user is away.
	IsAway bool `json:"is_away"`
	// AwayMessage is the user's AIM away message HTML.
	AwayMessage string `json:"away_message"`
	// IdleSeconds is the number of seconds the session has been idle.
	IdleSeconds int `json:"idle_seconds"`
	// IsInvisible is true when the user is invisible.
	IsInvisible bool `json:"is_invisible"`
	// IsICQ is true when the account is an ICQ user instead of an AIM user.
	IsICQ bool `json:"is_icq"`
	// InstanceCount is the number of concurrent clients signed in.
	InstanceCount int `json:"instance_count"`
	// Instances is the array of session instances for this user.
	Instances []SessionInstance `json:"instances"`
}

// SessionList is the response envelope returned by GET /session.
type SessionList struct {
	// Count is the number of active sessions.
	Count int `json:"count"`
	// Sessions is the list of active sessions.
	Sessions []Session `json:"sessions"`
}

// ChatParticipant describes a participant in a chat room.
type ChatParticipant struct {
	// ID is the participant's unique identifier.
	ID string `json:"id"`
	// ScreenName is the participant's AIM screen name.
	ScreenName string `json:"screen_name"`
}

// ChatRoom describes a chat room.
type ChatRoom struct {
	// Name is the name of the chat room.
	Name string `json:"name"`
	// CreateTime is the timestamp when the chat room was created.
	CreateTime time.Time `json:"create_time"`
	// CreatorID is the chat room creator user ID (private rooms only).
	CreatorID string `json:"creator_id,omitempty"`
	// URL is the AIM gochat URL for the room.
	URL string `json:"url"`
	// Participants is the list of participants in the chat room.
	Participants []ChatParticipant `json:"participants"`
}

// FeedbagBuddy describes a buddy within a feedbag group.
type FeedbagBuddy struct {
	// Name is the buddy's screen name.
	Name string `json:"name"`
	// ItemID is the feedbag item ID for this buddy.
	ItemID uint16 `json:"item_id"`
}

// FeedbagGroup describes a buddy list group.
type FeedbagGroup struct {
	// GroupID is the group ID.
	GroupID uint16 `json:"group_id"`
	// GroupName is the name of the group.
	GroupName string `json:"group_name"`
	// Buddies is the list of buddies in this group.
	Buddies []FeedbagBuddy `json:"buddies"`
}

// FeedbagGroupResult is returned when creating or fetching a feedbag group.
type FeedbagGroupResult struct {
	// GroupID is the group ID.
	GroupID uint16 `json:"group_id"`
	// GroupName is the name of the group.
	GroupName string `json:"group_name"`
}

// FeedbagBuddyResult is returned when adding a buddy to a group.
type FeedbagBuddyResult struct {
	// Name is the buddy's screen name.
	Name string `json:"name"`
	// GroupID is the group ID the buddy was added to.
	GroupID uint16 `json:"group_id"`
	// ItemID is the feedbag item ID for this buddy.
	ItemID uint16 `json:"item_id"`
}

// LinkedAccounts describes the linked accounts for a user.
type LinkedAccounts struct {
	// LinkedAccounts is the list of linked screen names.
	LinkedAccounts []string `json:"linked_accounts"`
}

// DirectoryCategory describes a keyword category in the directory.
type DirectoryCategory struct {
	// ID is the unique identifier of the keyword category.
	ID uint8 `json:"id"`
	// Name is the name of the keyword category.
	Name string `json:"name"`
}

// DirectoryKeyword describes a keyword in the directory.
type DirectoryKeyword struct {
	// ID is the unique identifier of the keyword.
	ID uint8 `json:"id"`
	// Name is the name of the keyword.
	Name string `json:"name"`
}

// WebAPIKey describes a Web API key used to authenticate against the Web AIM
// API. DevKey is only populated when a key is first created.
type WebAPIKey struct {
	// DevID is the unique developer/application identifier.
	DevID string `json:"dev_id"`
	// DevKey is the actual API key value, shown only at creation time.
	DevKey string `json:"dev_key,omitempty"`
	// AppName is the name of the application using this API key.
	AppName string `json:"app_name"`
	// CreatedAt is the timestamp when the key was created.
	CreatedAt time.Time `json:"created_at"`
	// LastUsed is the timestamp when the key was last used, if ever.
	LastUsed *time.Time `json:"last_used,omitempty"`
	// IsActive reports whether the API key is currently active.
	IsActive bool `json:"is_active"`
	// RateLimit is the maximum requests per minute allowed for this key.
	RateLimit int `json:"rate_limit"`
	// AllowedOrigins is the list of allowed CORS origins.
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
	// Capabilities is the list of enabled features for this key.
	Capabilities []string `json:"capabilities,omitempty"`
}

// CreateWebAPIKeyRequest is the request body for creating a Web API key.
type CreateWebAPIKeyRequest struct {
	// AppName is the name of the application using this API key (required).
	AppName string `json:"app_name"`
	// AllowedOrigins is the list of allowed CORS origins. An empty or nil
	// slice allows all origins.
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
	// RateLimit is the maximum requests per minute (defaults to 60 server-side
	// when zero or negative).
	RateLimit int `json:"rate_limit,omitempty"`
	// Capabilities is the list of capabilities enabled for this key. An empty
	// or nil slice allows all capabilities.
	Capabilities []string `json:"capabilities,omitempty"`
}

// UpdateWebAPIKeyRequest is the request body for updating a Web API key. Only
// fields with non-nil values are applied.
type UpdateWebAPIKeyRequest struct {
	// AppName, when non-nil, sets a new application name.
	AppName *string `json:"app_name,omitempty"`
	// IsActive, when non-nil, enables or disables the key.
	IsActive *bool `json:"is_active,omitempty"`
	// RateLimit, when non-nil, sets a new rate limit.
	RateLimit *int `json:"rate_limit,omitempty"`
	// AllowedOrigins, when non-nil, sets a new list of allowed origins.
	AllowedOrigins *[]string `json:"allowed_origins,omitempty"`
	// Capabilities, when non-nil, sets a new list of capabilities.
	Capabilities *[]string `json:"capabilities,omitempty"`
}

// PatchAccountRequest is the request body for patching a user account. Only
// fields with non-nil values are applied.
type PatchAccountRequest struct {
	// SuspendedStatus, when non-nil, sets the suspended status. Valid values
	// are "", "deleted", "expired", "suspended", and "suspended_age".
	SuspendedStatus *string `json:"suspended_status,omitempty"`
	// IsBot, when non-nil, sets the bot flag.
	IsBot *bool `json:"is_bot,omitempty"`
}
