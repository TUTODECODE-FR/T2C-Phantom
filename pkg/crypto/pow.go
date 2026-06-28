package crypto

import (
	"crypto/sha256"
	"fmt"
)

// TargetDifficulty defines the required number of leading zero bits for the Proof-of-Work.
// 21 bits typically takes ~1-3 seconds on a modern CPU to solve.
const TargetDifficulty = 21

// GeneratePoW brute-forces a nonce until sha256(challenge + nonce) has at least 'difficulty' leading zero bits.
func GeneratePoW(challenge string, difficulty int) string {
	var nonce uint64 = 0
	for {
		data := fmt.Sprintf("%s:%d", challenge, nonce)
		hash := sha256.Sum256([]byte(data))

		if hasLeadingZeroBits(hash[:], difficulty) {
			return fmt.Sprintf("%d", nonce)
		}
		nonce++
	}
}

// VerifyPoW checks if sha256(challenge + nonce) has at least 'difficulty' leading zero bits.
func VerifyPoW(challenge, nonceStr string, difficulty int) bool {
	data := fmt.Sprintf("%s:%s", challenge, nonceStr)
	hash := sha256.Sum256([]byte(data))
	return hasLeadingZeroBits(hash[:], difficulty)
}

// hasLeadingZeroBits checks if the first 'bits' bits of the byte array are zero.
func hasLeadingZeroBits(data []byte, bits int) bool {
	bytesToComplete := bits / 8
	remainingBits := bits % 8

	if len(data) < bytesToComplete {
		return false
	}

	for i := 0; i < bytesToComplete; i++ {
		if data[i] != 0 {
			return false
		}
	}

	if remainingBits > 0 {
		mask := byte(0xFF << (8 - remainingBits))
		if data[bytesToComplete]&mask != 0 {
			return false
		}
	}

	return true
}
