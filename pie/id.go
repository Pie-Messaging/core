package pie

import (
	"database/sql/driver"
	"golang.org/x/crypto/sha3"
	"math/big"
)

type ID big.Int
type IDA [IDLen]byte

func (i *ID) Scan(value any) error {
	(*big.Int)(i).SetBytes(value.([]byte))
	return nil
}

func (i ID) Value() (driver.Value, error) {
	return (*big.Int)(&i).Bytes(), nil
}

// HashBytes : max length of output is 128 bytes
func HashBytes(data []byte, hashLen int) []byte {
	h := make([]byte, hashLen)
	sha3.ShakeSum256(h, data)
	return h
}
