package types

// ServerProperties represents Minecraft server.properties key-value pairs.
type ServerProperties struct {
	EnableJmxMonitoring            string `json:"enable-jmx-monitoring,omitempty" binding:"omitempty,oneof=true false"`
	RconPort                       string `json:"rcon.port,omitempty" binding:"omitempty,number"`
	LevelSeed                      string `json:"level-seed,omitempty"`
	Gamemode                       string `json:"gamemode,omitempty" binding:"omitempty,oneof=survival creative adventure spectator"`
	EnableCommandBlock             string `json:"enable-command-block,omitempty" binding:"omitempty,oneof=true false"`
	EnableQuery                    string `json:"enable-query,omitempty" binding:"omitempty,oneof=true false"`
	GeneratorSettings              string `json:"generator-settings,omitempty"`
	EnforceSecureProfile           string `json:"enforce-secure-profile,omitempty" binding:"omitempty,oneof=true false"`
	LevelName                      string `json:"level-name,omitempty"`
	Motd                           string `json:"motd,omitempty"`
	QueryPort                      string `json:"query.port,omitempty" binding:"omitempty,number"`
	Pvp                            string `json:"pvp,omitempty" binding:"omitempty,oneof=true false"`
	GenerateStructures             string `json:"generate-structures,omitempty" binding:"omitempty,oneof=true false"`
	MaxChainedNeighborUpdates      string `json:"max-chained-neighbor-updates,omitempty" binding:"omitempty,number"`
	Difficulty                     string `json:"difficulty,omitempty" binding:"omitempty,oneof=peaceful easy normal hard"`
	NetworkCompressionThreshold    string `json:"network-compression-threshold,omitempty" binding:"omitempty,number"`
	MaxTickTime                    string `json:"max-tick-time,omitempty" binding:"omitempty,number"`
	RequireResourcePack            string `json:"require-resource-pack,omitempty" binding:"omitempty,oneof=true false"`
	UseNativeTransport             string `json:"use-native-transport,omitempty" binding:"omitempty,oneof=true false"`
	MaxPlayers                     string `json:"max-players,omitempty" binding:"omitempty,number"`
	OnlineMode                     string `json:"online-mode,omitempty" binding:"omitempty,oneof=true false"`
	EnableStatus                   string `json:"enable-status,omitempty" binding:"omitempty,oneof=true false"`
	AllowFlight                    string `json:"allow-flight,omitempty" binding:"omitempty,oneof=true false"`
	InitialDisabledPacks           string `json:"initial-disabled-packs,omitempty"`
	BroadcastRconToOps             string `json:"broadcast-rcon-to-ops,omitempty" binding:"omitempty,oneof=true false"`
	ViewDistance                   string `json:"view-distance,omitempty" binding:"omitempty,number"`
	ServerIp                       string `json:"server-ip,omitempty"`
	ResourcePackPrompt             string `json:"resource-pack-prompt,omitempty"`
	AllowNether                    string `json:"allow-nether,omitempty" binding:"omitempty,oneof=true false"`
	ServerPort                     string `json:"server-port,omitempty" binding:"omitempty,number"`
	EnableRcon                     string `json:"enable-rcon,omitempty" binding:"omitempty,oneof=true false"`
	SyncChunkWrites                string `json:"sync-chunk-writes,omitempty" binding:"omitempty,oneof=true false"`
	OpPermissionLevel              string `json:"op-permission-level,omitempty" binding:"omitempty,oneof=1 2 3 4"`
	PreventProxyConnections        string `json:"prevent-proxy-connections,omitempty" binding:"omitempty,oneof=true false"`
	HideOnlinePlayers              string `json:"hide-online-players,omitempty" binding:"omitempty,oneof=true false"`
	ResourcePack                   string `json:"resource-pack,omitempty"`
	EntityBroadcastRangePercentage string `json:"entity-broadcast-range-percentage,omitempty" binding:"omitempty,number"`
	SimulationDistance             string `json:"simulation-distance,omitempty" binding:"omitempty,number"`
	RconPassword                   string `json:"rcon.password,omitempty"`
	PlayerIdleTimeout              string `json:"player-idle-timeout,omitempty" binding:"omitempty,number"`
	ForceGamemode                  string `json:"force-gamemode,omitempty" binding:"omitempty,oneof=true false"`
	RateLimit                      string `json:"rate-limit,omitempty" binding:"omitempty,number"`
	Hardcore                       string `json:"hardcore,omitempty" binding:"omitempty,oneof=true false"`
	WhiteList                      string `json:"white-list,omitempty" binding:"omitempty,oneof=true false"`
	BroadcastConsoleToOps          string `json:"broadcast-console-to-ops,omitempty" binding:"omitempty,oneof=true false"`
	SpawnNpcs                      string `json:"spawn-npcs,omitempty" binding:"omitempty,oneof=true false"`
	SpawnAnimals                   string `json:"spawn-animals,omitempty" binding:"omitempty,oneof=true false"`
	LogIps                         string `json:"log-ips,omitempty" binding:"omitempty,oneof=true false"`
	FunctionPermissionLevel        string `json:"function-permission-level,omitempty" binding:"omitempty,oneof=1 2 3 4"`
	InitialEnabledPacks            string `json:"initial-enabled-packs,omitempty"`
	LevelType                      string `json:"level-type,omitempty"`
	TextFilteringConfig            string `json:"text-filtering-config,omitempty"`
	SpawnMonsters                  string `json:"spawn-monsters,omitempty" binding:"omitempty,oneof=true false"`
	EnforceWhitelist               string `json:"enforce-whitelist,omitempty" binding:"omitempty,oneof=true false"`
	SpawnProtection                string `json:"spawn-protection,omitempty" binding:"omitempty,number"`
	ResourcePackSha1               string `json:"resource-pack-sha1,omitempty"`
	MaxWorldSize                   string `json:"max-world-size,omitempty" binding:"omitempty,number"`
}

type StartServerRequest struct {
	CreateLaunchScript  bool             `json:"createLaunchScript"`
	ConfigureProperties bool             `json:"configureProperties"`
	Properties          ServerProperties `json:"properties"`
	ReleaseVersion      string           `json:"releaseVersion"`
}
