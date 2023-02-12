package app

import (
	"context"
	tbtypes "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
	"log"
)

type BalanceResponse struct {
	DebitsPosted   uint64
	CreditsPosted  uint64
	DebitsPending  uint64
	CreditsPending uint64
}

//encore:api public path=/balance/:accountId
func (s *Service) Balance(ctx context.Context, accountId string) (*BalanceResponse, error) {
	accountIdCasted, _ := tbtypes.HexStringToUint128(accountId)
	accounts, err := s.tbClient.LookupAccounts([]tbtypes.Uint128{accountIdCasted})
	if err != nil {
		log.Printf("Could not fetch accounts: %s", err)
		return nil, err
	}
	var account tbtypes.Account

	for _, acc := range accounts {
		account = acc
		break
	}
	return &BalanceResponse{
		DebitsPosted:   account.DebitsPosted,
		CreditsPosted:  account.CreditsPosted,
		DebitsPending:  account.DebitsPending,
		CreditsPending: account.CreditsPending,
	}, nil
}
