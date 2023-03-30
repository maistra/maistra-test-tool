package util

import (
	"fmt"
	"math/rand"
)

func NewNonce() string {
	return fmt.Sprintf("nonce-%d", rand.Int())
}
