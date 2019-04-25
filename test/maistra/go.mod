module github.com/yxun/moitt/maistra

require (
	istio.io/istio v0.0.0-20190425185518-e0a807d18fd9
	maistra/util v0.0.0
)

replace maistra/util v0.0.0 => ./util
