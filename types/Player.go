package types

type Player struct {
	UUID          string `json:"uuid"`
	Name          string `json:"name"`
	Online        bool   `json:"online"`
	IsOp          bool   `json:"is_op"`
	IsBanned      bool   `json:"is_banned"`
	IsWhitelisted bool   `json:"is_whitelisted"`
}

type UserCacheEntry struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	ExpiresOn string `json:"expiresOn"`
}

type OpEntry struct {
	UUID                string `json:"uuid"`
	Name                string `json:"name"`
	Level               int    `json:"level"`
	BypassesPlayerLimit bool   `json:"bypassesPlayerLimit"`
}

type BannedPlayerEntry struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Created string `json:"created"`
	Source  string `json:"source"`
	Expires string `json:"expires"`
	Reason  string `json:"reason"`
}

type WhitelistEntry struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// PlayerDeletionResult reports which of a DeletePlayer call's sub-actions
// actually took effect, since not all of them always apply (e.g. a player
// who was never op'd has nothing to deop).
type PlayerDeletionResult struct {
	Kicked           bool `json:"kicked"`
	Deopped          bool `json:"deopped"`
	Unwhitelisted    bool `json:"unwhitelisted"`
	UsercacheRemoved bool `json:"usercache_removed"`
}
