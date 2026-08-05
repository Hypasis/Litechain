package account

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// PasskeyAccount represents a biometric WebAuthn / Passkey-based Account (P-256)
type PasskeyAccount struct {
	Address  common.Address `json:"address"`
	PubKeyX  *big.Int       `json:"pubKeyX"`
	PubKeyY  *big.Int       `json:"pubKeyY"`
	KeyID    string         `json:"keyId"`
	Nonce    uint64         `json:"nonce"`
}

// NewPasskeyAccount creates a new Passkey account using a P-256 public key
func NewPasskeyAccount(keyID string, pubKeyX, pubKeyY *big.Int) *PasskeyAccount {
	// Generate deterministic account address from P-256 public key coordinates
	keyBytes := append(pubKeyX.Bytes(), pubKeyY.Bytes()...)
	addrHash := crypto.Keccak256(keyBytes)
	address := common.BytesToAddress(addrHash[12:])

	return &PasskeyAccount{
		Address:  address,
		PubKeyX:  pubKeyX,
		PubKeyY:  pubKeyY,
		KeyID:    keyID,
		Nonce:    0,
	}
}

// GeneratePasskeyKeyPair generates a test P-256 keypair (simulating Apple TouchID/FaceID enclave)
func GeneratePasskeyKeyPair() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// PasskeyVerifier handles WebAuthn secp256r1 (P-256) signature verification
type PasskeyVerifier struct {
	VerifiedCount uint64
}

// NewPasskeyVerifier creates a new PasskeyVerifier
func NewPasskeyVerifier() *PasskeyVerifier {
	return &PasskeyVerifier{}
}

// VerifyWebAuthnSignature verifies a WebAuthn P-256 signature produced by Apple FaceID / TouchID or Google Passkey
func (v *PasskeyVerifier) VerifyWebAuthnSignature(pubKeyX, pubKeyY *big.Int, authenticatorData, clientDataJSON []byte, r, s *big.Int) (bool, error) {
	if pubKeyX == nil || pubKeyY == nil || r == nil || s == nil {
		return false, fmt.Errorf("invalid public key or signature parameters")
	}

	// 1. Hash clientDataJSON using SHA-256
	clientDataHash := sha256.Sum256(clientDataJSON)

	// 2. Form signed payload: authenticatorData + clientDataHash
	signedPayload := append(authenticatorData, clientDataHash[:]...)

	// 3. Hash signed payload using SHA-256
	messageHash := sha256.Sum256(signedPayload)

	// 4. Construct P-256 ECDSA public key
	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     pubKeyX,
		Y:     pubKeyY,
	}

	// 5. Verify signature using P-256 curve
	valid := ecdsa.Verify(pubKey, messageHash[:], r, s)
	if !valid {
		return false, fmt.Errorf("Passkey P-256 signature verification failed")
	}

	v.VerifiedCount++
	return true, nil
}
