// Package uuid generates unique IDs for shapes and documents.
// Ported from MonoSketch uuid.
//
// Structure: version "-" datetime random random
// Length   :    2     1     7       10     10
//
// Characters: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
package uuid

import (
	"math"
	"math/rand/v2"
	"time"
)

const Version = "02"

const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

func Generate() string {
	now := math.Float64bits(float64(time.Now().UnixMilli()))
	datetime := toBase64(now)
	return Version + "-" + datetime[:len(datetime)-3] + toBase64(rand.Uint64()) + toBase64(rand.Uint64())
}

// toBase64 encodes the low 60 bits, least significant first.
func toBase64(number uint64) string {
	chars := make([]byte, 10)
	for i := range chars {
		chars[i] = base64Chars[number&0b111111]
		number >>= 6
	}
	return string(chars)
}
