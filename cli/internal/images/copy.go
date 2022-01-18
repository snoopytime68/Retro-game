package images

import (
	"github.com/defenseunicorns/zarf/cli/internal/message"
	"github.com/google/go-containerregistry/pkg/crane"
)

func Copy(src string, dest string) {
	if err := crane.Copy(src, dest, cranePlatformAMD64, cranePlatformARM64); err != nil {
		message.Fatal(err, "Unable to copy the image")
	}
}
