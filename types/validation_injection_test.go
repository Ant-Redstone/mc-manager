package types

import "testing"

// server.properties is line-oriented, so a newline in a value is not a
// formatting nuisance -- it appends settings the caller never asked for.
// These lock in the guard that makes that impossible.

func TestValidateServerProperties_RejectsNewlineInjection(t *testing.T) {
	// The concrete attack: smuggle a remote console onto the server via a
	// property whose value carries extra lines.
	err := ValidateServerProperties(map[string]string{
		"max-players": "10\nenable-rcon=true\nrcon.password=hunter2",
	})
	if err == nil {
		t.Fatal("expected newline-injected value to be rejected")
	}
}

func TestValidateServerProperties_RejectsCarriageReturnInjection(t *testing.T) {
	if err := ValidateServerProperties(map[string]string{"motd": "hi\r\nenable-rcon=true"}); err == nil {
		t.Fatal("expected CRLF-injected value to be rejected")
	}
}

func TestValidateServerProperties_RejectsBadKeys(t *testing.T) {
	if err := ValidateServerProperties(map[string]string{"": "x"}); err == nil {
		t.Error("expected an empty key to be rejected")
	}
	if err := ValidateServerProperties(map[string]string{"a=b": "x"}); err == nil {
		t.Error("expected '=' in a key to be rejected")
	}
	if err := ValidateServerProperties(map[string]string{"a\nb": "x"}); err == nil {
		t.Error("expected a newline in a key to be rejected")
	}
}

// The guard must apply to keys with no validation rule too -- those are
// precisely the ones the per-property rules skip, so without it an unknown
// key would be a free write primitive into the file.
func TestValidateServerProperties_GuardAppliesToUnknownKeys(t *testing.T) {
	if err := ValidateServerProperties(map[string]string{
		"some-unknown-property": "fine\ninjected=true",
	}); err == nil {
		t.Fatal("expected injection through an unrecognised key to be rejected")
	}
}

func TestValidateServerProperties_AllowsOrdinaryValues(t *testing.T) {
	err := ValidateServerProperties(map[string]string{
		"motd":         "A Minecraft Server - welcome!",
		"max-players":  "20",
		"difficulty":   "normal",
		"level-name":   "world",
		"unknown-prop": "some value with spaces & symbols =/-",
	})
	if err != nil {
		t.Errorf("expected ordinary values to pass, got: %v", err)
	}
}
