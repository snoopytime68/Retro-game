package utils

import (
	"os"

	"github.com/defenseunicorns/zarf/cli/internal/message"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
)

// DockerLogin adds the given creds to the user's Docker config, usually located at $HOME/.docker/config.yaml. It does not try
// to connect to the given registry, it just simply adds another entry to the config file.
// This function was mostly adapted from https://github.com/google/go-containerregistry/blob/5c9c442d5d68cd96787559ebf6e984c7eb084913/cmd/crane/cmd/auth.go
func DockerLogin(serverAddress string, user string, password string) error {
	message.Debugf("utils.DockerLogin(%s, %s, %s)", serverAddress, user, password)
	cf, err := config.Load(os.Getenv("DOCKER_CONFIG"))
	if err != nil {
		return err
	}
	creds := cf.GetCredentialsStore(serverAddress)
	if serverAddress == name.DefaultRegistry {
		serverAddress = authn.DefaultAuthKey
	}
	if err := creds.Store(types.AuthConfig{
		ServerAddress: serverAddress,
		Username:      user,
		Password:      password,
	}); err != nil {
		return err
	}

	if err := cf.Save(); err != nil {
		return err
	}
	message.Debugf("logged in via %s", cf.Filename)
	return nil
}
