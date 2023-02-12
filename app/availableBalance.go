package app

import (
	"context"
	"encore.dev/rlog"
	tbtypes "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
)

type AvailableBalanceResponse struct {
	AvailableBalance uint64
}

//encore:api public path=/available-balance/:accountId
func (s *Service) AvailableBalance(ctx context.Context, accountId string) (*AvailableBalanceResponse, error) {
	accountIdCasted, _ := tbtypes.HexStringToUint128(accountId)
	accounts, err := s.tbClient.LookupAccounts([]tbtypes.Uint128{accountIdCasted})
	if err != nil || len(accounts) == 0 {
		rlog.Error("failed to fetch accounts", "error", err, "accountId", accountId, "accountIdCasted", accountIdCasted)
		return nil, err
	}
	var account tbtypes.Account

	for _, acc := range accounts {
		account = acc
		break
	}
	return &AvailableBalanceResponse{
		AvailableBalance: account.CreditsPosted - account.DebitsPosted,
	}, nil
}
