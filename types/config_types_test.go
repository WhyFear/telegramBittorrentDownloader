package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestAPIConfigYAML(t *testing.T) {
	data := []byte(`
api:
  listen_ip: "0.0.0.0"
  port: 8081
  token: "secret"
`)
	var config Config

	err := yaml.Unmarshal(data, &config)

	require.NoError(t, err)
	assert.Equal(t, "0.0.0.0", config.API.ListenIP)
	assert.Equal(t, 8081, config.API.Port)
	assert.Equal(t, "secret", config.API.Token)
}
