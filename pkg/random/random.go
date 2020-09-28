package random

import (
	"crypto/rand"
	"math/big"
)

type RandomStringConfig struct {
	Length  int
	Charset string
}

func (r *RandomStringConfig) New() (string, error) {
	b := make([]byte, r.Length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(r.Charset))))
		if err != nil {
			return "", err
		}
		b[i] = r.Charset[n.Int64()]
	}
	return string(b), nil
}
