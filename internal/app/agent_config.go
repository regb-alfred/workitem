package app

import (
	"encoding/json"

	"github.com/regb/workitem/internal/config"
)

// driverConfigJSON applies agent overrides to driver defaults without changing
// either configured map. Conversation and auxiliary operations use the same merge.
func (a *App) driverConfigJSON(definition config.AgentDefinitionConfig) (json.RawMessage, error) {
	values := make(map[string]any)
	for key, value := range a.DriversConfig[definition.Driver].Defaults {
		values[key] = value
	}
	for key, value := range definition.Config {
		values[key] = value
	}
	return json.Marshal(values)
}
