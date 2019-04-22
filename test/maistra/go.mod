module github.com/yxun/moitt/maistra

require (
	gopkg.in/yaml.v2 v2.2.2
	istio.io/istio v0.0.0-20190422201318-a2c00674bfd7
	maistra/util v0.0.0
)

replace maistra/util v0.0.0 => ./util
