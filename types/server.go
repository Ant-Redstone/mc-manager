package types

// Server is a row in the servers registry: one Minecraft server instance
// this manager can operate on. Phase 1 of PLAN-multi-server.md always seeds
// exactly one row (see services.EnsureDefaultServer) pointing at the server
// directory that already existed before the registry did. Field order and
// json tags mirror the servers table in db/migrations.sql column for
// column.
type Server struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Dir       string `json:"dir"`
	Port      int    `json:"port"`
	VoicePort *int   `json:"voice_port"`
	Jar       string `json:"jar"`
	Xms       string `json:"xms"`
	Xmx       string `json:"xmx"`
	Sort      int    `json:"sort"`
	// CreatedAt is a plain string, not time.Time, matching how the existing
	// users.created_at TIMESTAMP column is already scanned in types.User --
	// see services/users.go's GetUsers.
	CreatedAt string `json:"created_at"`
}
