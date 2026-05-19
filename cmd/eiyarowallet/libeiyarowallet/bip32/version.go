package bip32

import "github.com/pkg/errors"

// BitcoinMainnetPrivate is the version that is used for
// bitcoin mainnet bip32 private extended keys.
// Ecnodes to xprv in base58.
var BitcoinMainnetPrivate = [4]byte{
	0x04,
	0x88,
	0xad,
	0xe4,
}

// BitcoinMainnetPublic is the version that is used for
// bitcoin mainnet bip32 public extended keys.
// Ecnodes to xpub in base58.
var BitcoinMainnetPublic = [4]byte{
	0x04,
	0x88,
	0xb2,
	0x1e,
}

// EiyaroMainnetPrivate is the version that is used for
// Eiyaro mainnet bip32 private extended keys.
// Ecnodes to xprv in base58.
var EiyaroMainnetPrivate = [4]byte{
	0x03,
	0x8f,
	0x2e,
	0xf4,
}

// EiyaroMainnetPublic is the version that is used for
// Eiyaro mainnet bip32 public extended keys.
// Ecnodes to kpub in base58.
var EiyaroMainnetPublic = [4]byte{
	0x03,
	0x8f,
	0x33,
	0x2e,
}

// EiyaroTestnetPrivate is the version that is used for
// Eiyaro testnet bip32 public extended keys.
// Ecnodes to ktrv in base58.
var EiyaroTestnetPrivate = [4]byte{
	0x03,
	0x90,
	0x9e,
	0x07,
}

// EiyaroTestnetPublic is the version that is used for
// Eiyaro testnet bip32 public extended keys.
// Ecnodes to ktub in base58.
var EiyaroTestnetPublic = [4]byte{
	0x03,
	0x90,
	0xa2,
	0x41,
}

// EiyaroDevnetPrivate is the version that is used for
// Eiyaro devnet bip32 public extended keys.
// Ecnodes to kdrv in base58.
var EiyaroDevnetPrivate = [4]byte{
	0x03,
	0x8b,
	0x3d,
	0x80,
}

// EiyaroDevnetPublic is the version that is used for
// Eiyaro devnet bip32 public extended keys.
// Ecnodes to xdub in base58.
var EiyaroDevnetPublic = [4]byte{
	0x03,
	0x8b,
	0x41,
	0xba,
}

// EiyaroSimnetPrivate is the version that is used for
// Eiyaro simnet bip32 public extended keys.
// Ecnodes to ksrv in base58.
var EiyaroSimnetPrivate = [4]byte{
	0x03,
	0x90,
	0x42,
	0x42,
}

// EiyaroSimnetPublic is the version that is used for
// Eiyaro simnet bip32 public extended keys.
// Ecnodes to xsub in base58.
var EiyaroSimnetPublic = [4]byte{
	0x03,
	0x90,
	0x46,
	0x7d,
}

func toPublicVersion(version [4]byte) ([4]byte, error) {
	switch version {
	case BitcoinMainnetPrivate:
		return BitcoinMainnetPublic, nil
	case EiyaroMainnetPrivate:
		return EiyaroMainnetPublic, nil
	case EiyaroTestnetPrivate:
		return EiyaroTestnetPublic, nil
	case EiyaroDevnetPrivate:
		return EiyaroDevnetPublic, nil
	case EiyaroSimnetPrivate:
		return EiyaroSimnetPublic, nil
	}

	return [4]byte{}, errors.Errorf("unknown version %x", version)
}

func isPrivateVersion(version [4]byte) bool {
	switch version {
	case BitcoinMainnetPrivate:
		return true
	case EiyaroMainnetPrivate:
		return true
	case EiyaroTestnetPrivate:
		return true
	case EiyaroDevnetPrivate:
		return true
	case EiyaroSimnetPrivate:
		return true
	}

	return false
}
