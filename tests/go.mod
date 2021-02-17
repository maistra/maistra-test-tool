module github.com/Maistra/maistra-test-tool

require (
	istio.io/pkg v0.0.0-20201112235759-c861803834b2
	maistra/util v0.0.0
)

replace maistra/util v0.0.0 => ./util

go 1.15
