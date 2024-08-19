// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	arch := env.GetArch()
	im, found := is[arch]
	if !found {
		panic(fmt.Sprintf("could not find image %q for architecture %q in %s", image, arch, yamlFile))
	}
	return im
}
