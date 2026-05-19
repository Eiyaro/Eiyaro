package libeiyarowallet

import (
	"encoding/binary"
	"encoding/hex"
	"math"

	"github.com/Eiyaro/Eiyaro/app/appmessage"
	"github.com/Eiyaro/Eiyaro/cmd/eiyarowallet/daemon/pb"
	"github.com/Eiyaro/Eiyaro/domain/consensus/model/externalapi"
	"github.com/Eiyaro/Eiyaro/domain/consensus/utils/transactionid"
	"github.com/Eiyaro/Eiyaro/domain/consensus/utils/utxo"
	"github.com/pkg/errors"
)

func checkedVersionUint16(version uint32) uint16 {
	var versionBytes [4]byte
	binary.BigEndian.PutUint32(versionBytes[:], version)
	return binary.BigEndian.Uint16(versionBytes[2:])
}

// EiyarowalletdUTXOsTolibeiyarowalletUTXOs converts a  []*pb.UtxosByAddressesEntry to a []*libeiyarowallet.UTXO
func EiyarowalletdUTXOsTolibeiyarowalletUTXOs(eiyarowalletdUtxoEntires []*pb.UtxosByAddressesEntry) ([]*UTXO, error) {
	UTXOs := make([]*UTXO, len(eiyarowalletdUtxoEntires))
	for i, entry := range eiyarowalletdUtxoEntires {
		script, err := hex.DecodeString(entry.UtxoEntry.ScriptPublicKey.ScriptPublicKey)
		if err != nil {
			return nil, err
		}
		transactionID, err := transactionid.FromString(entry.Outpoint.TransactionId)
		if err != nil {
			return nil, err
		}
		if entry.UtxoEntry.ScriptPublicKey.Version > math.MaxUint16 {
			return nil, errors.Errorf("script public key version %d exceeds uint16", entry.UtxoEntry.ScriptPublicKey.Version)
		}
		UTXOs[i] = &UTXO{
			UTXOEntry: utxo.NewUTXOEntry(
				entry.UtxoEntry.Amount,
				&externalapi.ScriptPublicKey{
					Script:  script,
					Version: checkedVersionUint16(entry.UtxoEntry.ScriptPublicKey.Version),
				},
				entry.UtxoEntry.IsCoinbase,
				entry.UtxoEntry.BlockDaaScore,
			),
			Outpoint: &externalapi.DomainOutpoint{
				TransactionID: *transactionID,
				Index:         entry.Outpoint.Index,
			},
		}
	}
	return UTXOs, nil
}

// AppMessageUTXOToEiyarowalletdUTXO converts an appmessage.UTXOsByAddressesEntry to a  pb.UtxosByAddressesEntry
func AppMessageUTXOToEiyarowalletdUTXO(appUTXOsByAddressesEntry *appmessage.UTXOsByAddressesEntry) *pb.UtxosByAddressesEntry {
	return &pb.UtxosByAddressesEntry{
		Outpoint: &pb.Outpoint{
			TransactionId: appUTXOsByAddressesEntry.Outpoint.TransactionID,
			Index:         appUTXOsByAddressesEntry.Outpoint.Index,
		},
		UtxoEntry: &pb.UtxoEntry{
			Amount: appUTXOsByAddressesEntry.UTXOEntry.Amount,
			ScriptPublicKey: &pb.ScriptPublicKey{
				Version:         uint32(appUTXOsByAddressesEntry.UTXOEntry.ScriptPublicKey.Version),
				ScriptPublicKey: appUTXOsByAddressesEntry.UTXOEntry.ScriptPublicKey.Script,
			},
			BlockDaaScore: appUTXOsByAddressesEntry.UTXOEntry.BlockDAAScore,
			IsCoinbase:    appUTXOsByAddressesEntry.UTXOEntry.IsCoinbase,
		},
	}
}
