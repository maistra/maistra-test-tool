module github.com/Maistra/maistra-test-tool

require (
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	istio.io/pkg v0.0.0-20200422223412-3fdcd0a1c360
	maistra/util v0.0.0
)

replace maistra/util v0.0.0 => ./util

go 1.13
