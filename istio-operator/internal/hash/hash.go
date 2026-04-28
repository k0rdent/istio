package hash

import (
	"fmt"
	"hash/adler32"
	"hash/fnv"
)

type HashType int

const (
	// FnvHash is FNV-1a 32-bit - fast, non-cryptographic hash with good distribution.
	FnvHash HashType = iota
	// AdlerHash is Adler-32 checksum. Matches Helm's adler32sum template function.
	AdlerHash
)

// WithPrefix returns "prefix-hash" string using the selected hash algorithm.
// Hash is returned as lowercase hex.
func WithPrefix(prefix, data string, ht HashType) string {
	fn := HashFunc(ht)
	h := fn(data)
	return fmt.Sprintf("%s-%s", prefix, h)
}

// HashFunc returns the appropriate hashing function for the given HashType.
func HashFunc(hashType HashType) func(string) string {
	switch hashType {
	case FnvHash:
		return Fnv32a
	case AdlerHash:
		return Adler32sum
	default:
		return Adler32sum
	}
}

// Fnv32a computes FNV-1a 32-bit hash of the string.
func Fnv32a(str string) string {
	h := fnv.New32a()
	h.Write([]byte(str))
	return fmt.Sprintf("%x", h.Sum32())
}

// Adler32sum computes Adler-32 checksum of the string.
func Adler32sum(str string) string {
	return fmt.Sprintf("%d", adler32.Checksum([]byte(str)))
}
