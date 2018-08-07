package crypto

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	abci "github.com/tendermint/tendermint/abci/types"
	tmCrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/tmhash"
	"github.com/tmthrgd/go-hex"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ripemd160"
)

type PublicKeyJSON struct {
	CurveType string
	PublicKey string
}

// Returns the length in bytes of the public key
func PublicKeyLength(curveType CurveType) int {
	switch curveType {
	case CurveTypeEd25519:
		return ed25519.PublicKeySize
	case CurveTypeSecp256k1:
		return btcec.PubKeyBytesLenCompressed
	default:
		// Other functions rely on this
		return 0
	}
}

func (p PublicKey) IsSet() bool {
	return p.CurveType != CurveTypeUnset && p.IsValid()
}

func (p PublicKey) MarshalJSON() ([]byte, error) {
	jStruct := PublicKeyJSON{
		CurveType: p.CurveType.String(),
		PublicKey: hex.EncodeUpperToString(p.Key),
	}
	txt, err := json.Marshal(jStruct)
	return txt, err
}

func (p PublicKey) MarshalText() ([]byte, error) {
	return p.MarshalJSON()
}

func (p *PublicKey) UnmarshalJSON(text []byte) error {
	var jStruct PublicKeyJSON
	err := json.Unmarshal(text, &jStruct)
	if err != nil {
		return err
	}
	CurveType, err := CurveTypeFromString(jStruct.CurveType)
	if err != nil {
		return err
	}
	bs, err := hex.DecodeString(jStruct.PublicKey)
	if err != nil {
		return err
	}
	p.CurveType = CurveType
	p.Key = bs
	return nil
}

func (p *PublicKey) UnmarshalText(text []byte) error {
	return p.UnmarshalJSON(text)
}

func (p PublicKey) IsValid() bool {
	publicKeyLength := PublicKeyLength(p.CurveType)
	return publicKeyLength != 0 && publicKeyLength == len(p.Key)
}

func (p PublicKey) Verify(msg []byte, signature Signature) error {
	switch p.CurveType {
	case CurveTypeUnset:
		return fmt.Errorf("public key is unset")
	case CurveTypeEd25519:
		if ed25519.Verify(p.Key, msg, signature) {
			return nil
		}
		return fmt.Errorf("'%X' is not a valid ed25519 signature for message: %X", signature, msg)
	case CurveTypeSecp256k1:
		pub, err := btcec.ParsePubKey(p.Key, btcec.S256())
		if err != nil {
			return fmt.Errorf("could not parse secp256k1 public key: %v", err)
		}
		sig, err := btcec.ParseDERSignature(signature, btcec.S256())
		if err != nil {
			return fmt.Errorf("could not parse DER signature for secp256k1 key: %v", err)
		}
		if sig.Verify(msg, pub) {
			return nil
		}
		return fmt.Errorf("'%X' is not a valid secp256k1 signature for message: %X", signature, msg)
	default:
		return fmt.Errorf("invalid curve type")
	}
}

func (p PublicKey) PublicKey() PublicKey {
	return p
}

func (p PublicKey) Address() Address {
	switch p.CurveType {
	case CurveTypeEd25519:
		addr, _ := AddressFromBytes(tmhash.Sum(p.Key))
		return addr
	case CurveTypeSecp256k1:
		sha := sha256.New()
		sha.Write(p.Key[:])

		hash := ripemd160.New()
		hash.Write(sha.Sum(nil))
		addr, _ := AddressFromBytes(hash.Sum(nil))
		return addr
	default:
		panic(fmt.Sprintf("unknown CurveType %d", p.CurveType))
	}
}

func (p PublicKey) AddressHashType() string {
	switch p.CurveType {
	case CurveTypeEd25519:
		return "go-crypto-0.5.0"
	case CurveTypeSecp256k1:
		return "btc"
	default:
		return ""
	}
}

func (p PublicKey) RawBytes() []byte {
	return p.Key[:]
}

// Return the ABCI PubKey. See Tendermint protobuf.go for the go-crypto conversion this is based on
func (p PublicKey) ABCIPubKey() abci.PubKey {
	return abci.PubKey{
		Type: p.CurveType.ABCIType(),
		Data: p.RawBytes(),
	}
}

func PublicKeyFromTendermintPubKey(pubKey tmCrypto.PubKey) (PublicKey, error) {
	switch pk := pubKey.(type) {
	case tmCrypto.PubKeyEd25519:
		return PublicKeyFromBytes(pk[:], CurveTypeEd25519)
	case tmCrypto.PubKeySecp256k1:
		return PublicKeyFromBytes(pk[:], CurveTypeSecp256k1)
	default:
		return PublicKey{}, fmt.Errorf("unrecognised tendermint public key type: %v", pk)
	}

}
func PublicKeyFromABCIPubKey(pubKey abci.PubKey) (PublicKey, error) {
	switch pubKey.Type {
	case CurveTypeEd25519.ABCIType():
		return PublicKey{
			CurveType: CurveTypeEd25519,
			Key:       pubKey.Data,
		}, nil
	case CurveTypeSecp256k1.ABCIType():
		return PublicKey{
			CurveType: CurveTypeEd25519,
			Key:       pubKey.Data,
		}, nil
	}
	return PublicKey{}, fmt.Errorf("did not recognise ABCI PubKey type: %s", pubKey.Type)
}

func (p PublicKey) String() string {
	return hex.EncodeUpperToString(p.Key)
}

// Produces a binary encoding of the CurveType byte plus
// the public key for padded to a fixed width on the right
func (p PublicKey) Encode() []byte {
	encoded := make([]byte, PublicKeyLength(p.CurveType)+1)
	encoded[0] = p.CurveType.Byte()
	copy(encoded[1:], p.Key)
	return encoded
}
