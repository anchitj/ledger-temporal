package app

import (
	"context"
	"encore.dev/rlog"
	tbtypes "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
)

type AccountResponse struct {
	AccountId string
	Amount    uint64
}

//encore:api public method=POST path=/account/:accountId
func (s *Service) Account(ctx context.Context, accountId string) (*AccountResponse, error) {
	accountIdCasted, _ := tbtypes.HexStringToUint128(accountId)
	account := tbtypes.Account{
		ID:             accountIdCasted,
		Ledger:         LedgerId,
		Code:           LedgerId,
		UserData:       accountIdCasted,
		Reserved:       [48]uint8{},
		Flags:          tbtypes.AccountFlags{}.ToUint16(),
		DebitsPending:  0,
		DebitsPosted:   0,
		CreditsPending: 0,
		CreditsPosted:  0,
		Timestamp:      0,
	}
	res, err := s.tbClient.CreateAccounts([]tbtypes.Account{account})
	if err != nil {
		rlog.Error("failed to create account", "error", err)
		return nil, err
	}
	for _, r := range res {
		rlog.Info("result " + r.Result.String())
	}
	return &AccountResponse{
		AccountId: accountId,
		Amount:    0,
	}, nil
}
