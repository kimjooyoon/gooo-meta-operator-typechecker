package meta

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func DigestBytes(raw []byte) string {
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

func DigestValue(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return DigestBytes(raw), nil
}

func unsignedIRDigest(ir IR) (string, error) {
	ir.Proofs = append([]Proof(nil), ir.Proofs...)
	ir.IRDigest = ""
	for index := range ir.Proofs {
		ir.Proofs[index].IRDigest = ""
	}
	return DigestValue(ir)
}
