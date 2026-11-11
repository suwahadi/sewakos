package midtrans

import (
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
)

func VerifyNotificationSignature(serverKey, orderID, statusCode, grossAmount, signatureKey string) bool {
	sum := sha512.Sum512([]byte(orderID + statusCode + grossAmount + serverKey))
	expected := hex.EncodeToString(sum[:])
	// Midtrans-supplied signatures must be compared without leaking timing
	// information about how many leading bytes matched.
	return subtle.ConstantTimeCompare([]byte(expected), []byte(signatureKey)) == 1
}
