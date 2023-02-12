package workflow

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	tb "github.com/tigerbeetledb/tigerbeetle-go"
	tbtypes "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
	"log"
	"strconv"
	"time"
)

type Activities struct {
	RedisClient *redis.Client
	TbClient    tb.Client
}

func generateTransferId(accountId tbtypes.Uint128) tbtypes.Uint128 {
	transferId, err := tbtypes.HexStringToUint128(strconv.FormatInt(time.Now().Unix(), 10))
	if err != nil {
		log.Printf("Could not generate transfer id: %s", err)
	}
	//log.Printf("Transfer id generated for account %s: %s", accountId, transferId)
	return transferId
}

func storeAuthorizationRedis(debitAccountId tbtypes.Uint128, amount uint64, transferId tbtypes.Uint128, redisClient *redis.Client) {
	redisClient.RPush("authorizations:"+debitAccountId.String()+":amounts:"+strconv.Itoa(int(amount))+":transfers", transferId.String())
}

func getAuthorizationRedis(debitAccountId tbtypes.Uint128, amount uint64, redisClient *redis.Client) ([]tbtypes.Uint128, error) {
	key := fmt.Sprintf("authorizations:%s:amounts:%d:transfers", debitAccountId, amount)
	log.Printf("Checking authorizations for Key: %s", key)
	transfers, err := redisClient.LRange(key, 0, -1).Result()
	if err != nil {
		log.Printf("Could not get authorization from redis: %s", err)
		return nil, err
	}
	var returnTransfers []tbtypes.Uint128
	for _, transfer := range transfers {
		tid, _ := tbtypes.HexStringToUint128(transfer)
		returnTransfers = append(returnTransfers, tid)
	}
	return returnTransfers, nil
}

func voidAuthorization(transferId tbtypes.Uint128, tbClient tb.Client) error {
	transfer := tbtypes.Transfer{
		ID: generateTransferId(transferId),
		Flags: tbtypes.TransferFlags{
			VoidPendingTransfer: true,
		}.ToUint16(),
		PendingID: transferId,
	}
	res, err := tbClient.CreateTransfers([]tbtypes.Transfer{transfer})
	if err != nil {
		log.Printf("Error creating transfer batch %s", err)
		return err
	}
	for _, t := range res {
		id := int(t.Index)
		log.Printf("Transfer %s created %d : id", t.Result, id)
	}
	return nil
}

func removeVoidAuthorizationRedis(debitAccountId tbtypes.Uint128, amount uint64, transferId tbtypes.Uint128, redisClient *redis.Client) error {
	key := fmt.Sprintf("authorizations:%s:amounts:%d:transfers", debitAccountId, amount)
	err := redisClient.LRem(key, 0, transferId.String()).Err()
	if err != nil {
		log.Printf("Could not remove authorization from redis: %s", err)
		return err
	}
	return nil
}

func postPendingAuthorization(accountId tbtypes.Uint128, pendingId tbtypes.Uint128, tbClient tb.Client) error {
	transfer := tbtypes.Transfer{
		ID:        generateTransferId(accountId),
		PendingID: pendingId,
		Flags: tbtypes.TransferFlags{
			PostPendingTransfer: true,
		}.ToUint16(),
	}
	res, err := tbClient.CreateTransfers([]tbtypes.Transfer{transfer})
	if err != nil {
		log.Printf("Error creating transfer batch %s", err)
		return err
	}
	for _, t := range res {
		log.Printf("Transfer %s created", t.Result)
	}
	return nil
}

func (a *Activities) CheckAccountExists(_ context.Context, accountId tbtypes.Uint128) (bool, error) {
	accounts, err := a.TbClient.LookupAccounts([]tbtypes.Uint128{accountId})
	if err != nil {
		log.Printf("Could not fetch accounts: %s", err)
		return false, err
	}
	if len(accounts) == 0 {
		return false, nil
	}
	return true, nil
}

func (a *Activities) CheckAccountExistsWithSufficientBalance(_ context.Context, accountId tbtypes.Uint128, amount uint64) (bool, error) {
	accounts, err := a.TbClient.LookupAccounts([]tbtypes.Uint128{accountId})
	if err != nil {
		log.Printf("Could not fetch accounts: %s", err)
		return false, err
	}
	if len(accounts) == 0 {
		return false, nil
	}
	var account tbtypes.Account
	for _, acc := range accounts {
		account = acc
		break
	}
	if account.CreditsPosted-account.DebitsPosted < amount {
		return false, errors.New("insufficient balance")
	}
	return true, nil
}

func (a *Activities) PlaceAuthorization(_ context.Context, debitAccountId tbtypes.Uint128, creditAccountId tbtypes.Uint128, amount uint64) (tbtypes.Uint128, error) {
	transfer := tbtypes.Transfer{
		ID:              generateTransferId(debitAccountId),
		DebitAccountID:  debitAccountId,
		CreditAccountID: creditAccountId,
		Amount:          amount,
		Flags: tbtypes.TransferFlags{
			Pending: true,
		}.ToUint16(),
		Timeout: uint64(AuthorizationHoldDuration.Nanoseconds()),
		Ledger:  1,
		Code:    1,
	}
	res, err := a.TbClient.CreateTransfers([]tbtypes.Transfer{transfer})
	if err != nil {
		log.Printf("Error creating transfer batch %s", err)
		return InvalidTransferId, err
	}
	for _, t := range res {
		id := int(t.Index)
		log.Printf("Transfer %s created %d : id", t.Result, id)
	}
	storeAuthorizationRedis(debitAccountId, amount, transfer.ID, a.RedisClient)
	return transfer.ID, nil
}

func (a *Activities) MatchPresentment(_ context.Context, debitAccountId tbtypes.Uint128, amount uint64) (tbtypes.Uint128, error) {
	authorizations, err := getAuthorizationRedis(debitAccountId, amount, a.RedisClient)
	log.Printf("got authorizations: %s", authorizations)
	if err != nil {
		log.Printf("Could not get authorization from redis: %s", err)
		return InvalidTransferId, err
	}
	if len(authorizations) == 0 {
		log.Printf("no authorizations found")
		return InvalidTransferId, nil
	}

	transfers, err := a.TbClient.LookupTransfers(authorizations)
	if err != nil {
		log.Printf("Could not fetch transfers: %s", err)
		return InvalidTransferId, err
	}
	log.Printf("got transfers from tb: %s", transfers)
	pendingFlag := tbtypes.TransferFlags{Pending: true}.ToUint16()
	for _, transfer := range transfers {
		log.Printf("checking for transfer: %s", transfer)
		if transfer.Flags == pendingFlag && transfer.Timestamp+transfer.Timeout > uint64(time.Now().UnixNano()) {
			log.Printf("pending transfer: %s", transfer)
			return transfer.ID, nil
		} else {
			err = voidAuthorization(transfer.ID, a.TbClient)
			if err != nil {
				log.Printf("Could not void pending auth: %s", err)
			}
			err = removeVoidAuthorizationRedis(debitAccountId, amount, transfer.ID, a.RedisClient)
			if err != nil {
				log.Printf("Could not remove authorization from redis: %s", err)
			}
		}
	}
	return InvalidTransferId, nil
}

func (a *Activities) PostPendingTransfer(_ context.Context, transferId, debitAccountId tbtypes.Uint128, amount uint64) error {
	err := postPendingAuthorization(debitAccountId, transferId, a.TbClient)
	if err != nil {
		log.Printf("Error in postPendingAuthorization: %s for transfer: %s, continuing", err, transferId)
		return err
	}
	err = removeVoidAuthorizationRedis(debitAccountId, amount, transferId, a.RedisClient)
	if err != nil {
		log.Printf("Could not remove authorization from redis: %s", err)
	}
	return nil
}
