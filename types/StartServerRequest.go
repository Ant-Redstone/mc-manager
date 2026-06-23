package types

type StartServerRequest struct {
	CreateLaunchScript  bool              `json:"createLaunchScript"`
	ConfigureProperties bool              `json:"configureProperties"`
	Properties          map[string]string `json:"properties"`
	ReleaseVersion      string            `json:"releaseVersion"`
}
