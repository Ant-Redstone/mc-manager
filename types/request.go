package types

type CreateServerRequest struct {
	ServerType          string            `json:"serverType" binding:"required"`
	ReleaseVersion      string            `json:"releaseVersion"`
	LoaderVersion       string            `json:"loaderVersion"`
	CreateLaunchScript  bool              `json:"createLaunchScript"`
	ConfigureProperties bool              `json:"configureProperties"`
	Properties          map[string]string `json:"properties"`
}

type StartServerRequest struct {
}

type UpdateServerPropertiesRequest struct {
	Properties map[string]string `json:"properties"`
}

type RegisterRequest struct {
	Token    string `json:"token" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UpdateProfileRequest carries a caller's edits to their own profile.
// DisplayName is the only field so far; an empty string clears it back to
// falling through to Username on display.
type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
}
