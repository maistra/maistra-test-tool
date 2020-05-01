module github.com/Maistra/maistra-test-tool

require (
	istio.io/pkg v0.0.0-20200422223412-3fdcd0a1c360
	maistra/util v0.0.0
)

replace maistra/util v0.0.0 => ./util

go 1.13
