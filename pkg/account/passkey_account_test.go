package account

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

func TestPasskeySignatureVerification(t *testing.T) {
	verifier := NewPasskeyVerifier()

	// 1. Generate P-256 Passkey keypair (simulating Apple TouchID / FaceID enclave)
	privateKey, err := GeneratePasskeyKeyPair()
	if err != nil {
		t.Fatalf("failed to generate Passkey keypair: %v", err)
	}

	account := NewPasskeyAccount("passkey-user-iphone-16", privateKey.PublicKey.X, privateKey.PublicKey.Y)
	if account.Address == (common.Address{}) {
		t.Errorf("invalid passkey account address")
	}

	authenticatorData := []byte("apple_biometric_authenticator_data_32bytes")
	clientDataJSON := []byte(`{"type":"webauthn.get","challenge":"litechain_passkey_challenge","origin":"https://lightchain.l1"}`)

	clientDataHash := sha256.Sum256(clientDataJSON)
	signedPayload := append(authenticatorData, clientDataHash[:]...)
	messageHash := sha256.Sum256(signedPayload)

	// Sign payload with P-256 private key
	r, s, err := ecdsa.Sign(randReader{}, privateKey, messageHash[:])
	if err != nil {
		t.Fatalf("failed to sign Passkey payload: %v", err)
	}

	// 2. Verify signature using P-256 curve
	valid, err := verifier.VerifyWebAuthnSignature(privateKey.PublicKey.X, privateKey.PublicKey.Y, authenticatorData, clientDataJSON, r, s)
	if err != nil || !valid {
		t.Fatalf("Passkey verification failed: valid=%v, err=%v", valid, err)
	}
}

// Simple randReader helper for deterministic testing
type randReader struct{}

func (r randReader) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte(i + 1)
	}
	return len(p), nil
}

func TestERC4337UserOperationBundler(t *testing.T) {
	entryPoint := common.HexToAddress("0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789")
	chainID := big.NewInt(1337)

	bundler := NewBundlerEngine(entryPoint, chainID)

	key, _ := crypto.GenerateKey()
	senderAddr := crypto.PubkeyToAddress(key.PublicKey)

	// 1. Create UserOperation with paymaster gas sponsorship
	paymasterData := append(entryPoint.Bytes(), []byte("sponsorship_policy_active")...)
	userOp := &UserOperation{
		Sender:               senderAddr,
		Nonce:                uint256.NewInt(0),
		InitCode:             nil,
		CallData:             []byte("0xa9059cbb000000000000000000000000"), // ERC20 transfer selector
		CallGasLimit:         100000,
		VerificationGasLimit: 50000,
		PreVerificationGas:   21000,
		MaxFeePerGas:         uint256.NewInt(1000000000),
		MaxPriorityFeePerGas: uint256.NewInt(100000000),
		PaymasterAndData:     paymasterData,
	}

	// Sign UserOperation
	err := userOp.Sign(entryPoint, chainID, key)
	if err != nil {
		t.Fatalf("failed to sign UserOp: %v", err)
	}

	// 2. Validate UserOperation in bundler
	valid, err := bundler.ValidateUserOperation(userOp)
	if err != nil || !valid {
		t.Fatalf("UserOp validation failed: valid=%v, err=%v", valid, err)
	}

	// 3. Bundle operations
	bundle := bundler.BundleUserOperations(10)
	if len(bundle) != 1 {
		t.Fatalf("expected bundle size of 1, got %d", len(bundle))
	}
}
