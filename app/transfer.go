package app

import (
	"context"
	"encore.dev/rlog"
	tbtypes "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
	"strconv"
	"time"
)

type TransferResponse struct {
	DebitAccountId  string
	CreditAccountId string
	Amount          uint64
}

//encore:api public method=POST path=/transfer/:debitAccountId/:creditAccountId/:amount
func (s *Service) Transfer(ctx context.Context, debitAccountId string, creditAccountId string, amount uint64) (*TransferResponse, error) {
	debitAccountIdCasted, _ := tbtypes.HexStringToUint128(debitAccountId)
	creditAccountIdCasted, _ := tbtypes.HexStringToUint128(creditAccountId)
	transferId, _ := tbtypes.HexStringToUint128(strconv.FormatInt(time.Now().Unix(), 10))

	transfer := tbtypes.Transfer{
		ID:              transferId,
		DebitAccountID:  debitAccountIdCasted,
		CreditAccountID: creditAccountIdCasted,
		Amount:          amount,
		Ledger:          1,
		Code:            1,
	}

	res, err := s.tbClient.CreateTransfers([]tbtypes.Transfer{transfer})
	if err != nil {
		rlog.Error("failed to create transfer", "error", err)
		return nil, err
	}

	for _, r := range res {
		rlog.Info("result " + r.Result.String())
	}

	return &TransferResponse{
		DebitAccountId:  debitAccountId,
		CreditAccountId: creditAccountId,
		Amount:          amount,
	}, nil
}

//encore:api public method=GET path=/transfer/:transferId
func (s *Service) GetTransfer(ctx context.Context, transferId string) (*TransferResponse, error) {
	transferIdCasted, _ := tbtypes.HexStringToUint128(transferId)
	var transfers []tbtypes.Uint128
	transfers = append(transfers, transferIdCasted)

	transfer, err := s.tbClient.LookupTransfers(transfers)
	if err != nil {
		rlog.Error("failed to get transfer", "error", err)
		return nil, err
	}

	if len(transfer) == 0 {
		rlog.Info("transfer not found")
		return nil, nil
	}

	rlog.Info("transfer found " + transfer[0].ID.String() + " " + strconv.Itoa(int(transfer[0].Flags)) + " " + strconv.Itoa(int(transfer[0].Timestamp)))

	return nil, nil
}
