package template

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
)

// imageMap is initialized from the images.yaml file and
// maps image -> (architecture -> container image)
var imageMap map[string]map[string]string

var yamlFile = env.GetRootDir() + "/images.yaml"

func init() {
	imageMap = map[string]map[string]string{}
	bytes, err := os.ReadFile(yamlFile)
	if err != nil {
		panic(fmt.Sprintf("couldn't read file %s: %v", yamlFile, err))
	}

	err = yaml.Unmarshal(bytes, &imageMap)
	if err != nil {
		panic(fmt.Sprintf("couldn't parse file %s: %v", yamlFile, err))
	}
}

// image returns the correct container image for the current architecture
func image(image string) string {
	is, found := imageMap[image]
	if !found {
		panic(fmt.Sprintf("could not find image %q in %s", image, yamlFile))
	}

	arch := env.Getenv("SAMPLEARCH", "x86")
	im, found := is[arch]
	if !found {
		panic(fmt.Sprintf("could not find image %q for architecture %q in %s", image, arch, yamlFile))
	}
	return im
}
