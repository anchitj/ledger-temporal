package workflow

import (
	"github.com/go-redis/redis"
	tbtypes "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
	"go.temporal.io/sdk/workflow"
	"log"
	"time"
)

const (
	CreditAccountId           = "1234567"
	AuthorizationHoldDuration = 1 * time.Minute
	InvalidTransferIdString   = "00000000000000000000000000000000"
)

var InvalidTransferId, _ = tbtypes.HexStringToUint128(InvalidTransferIdString)

func Auth(ctx workflow.Context, accountId tbtypes.Uint128, amount uint64, redisClient *redis.Client) (bool, error) {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
	}

	ctx = workflow.WithActivityOptions(ctx, options)

	var accountExists bool
	err := workflow.ExecuteActivity(ctx, CheckAccountExistsWithSufficientBalance, accountId, amount).Get(ctx, &accountExists)
	if err != nil {
		log.Printf("Could not check account existence: %s", err)
		return false, err
	}
	if !accountExists {
		log.Printf("Account %s does not exist", accountId)
		return false, nil
	}

	creditAccountIdCasted, _ := tbtypes.HexStringToUint128(CreditAccountId)
	log.Printf("starting authorization for %d on account %s %s %s", amount, accountId, accountId.String(), creditAccountIdCasted.String())

	var transferId tbtypes.Uint128
	err = workflow.ExecuteActivity(ctx, PlaceAuthorization, accountId, creditAccountIdCasted, amount, redisClient).Get(ctx, &transferId)
	if err != nil {
		log.Printf("Could not place authoriation: %s", err)
		return false, err
	}

	log.Printf("Placed authorization for %d on account %s", amount, accountId)
	//time.Sleep(AuthorizationHoldDuration)
	//
	//var isAuthorizationPending bool
	//err = workflow.ExecuteActivity(ctx, IsAuthorizationPending, transferId).Get(ctx, &isAuthorizationPending)
	//if err != nil {
	//	log.Printf("Could not check status: %s", err)
	//	return false, err
	//}
	//
	//if isAuthorizationPending {
	//	err = workflow.ExecuteActivity(ctx, VoidAuthorization, transferId, redisClient).Get(ctx, nil)
	//	if err != nil {
	//		log.Printf("Could not void authorization: %s", err)
	//		return false, err
	//	}
	//}
	return true, nil
}

func Present(ctx workflow.Context, accountId tbtypes.Uint128, amount uint64, redisClient *redis.Client) (bool, error) {
	//retrypolicy := &temporal.RetryPolicy{
	//	MaximumAttempts: 1,
	//}
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
		//RetryPolicy:         retrypolicy,
	}

	ctx = workflow.WithActivityOptions(ctx, options)

	var accountExists bool
	err := workflow.ExecuteActivity(ctx, CheckAccountExists, accountId).Get(ctx, &accountExists)
	if err != nil {
		log.Printf("Could not check account existence: %s", err)
		return false, err
	}
	if !accountExists {
		log.Printf("Account %s does not exist", accountId)
		return false, nil
	}

	var transferId tbtypes.Uint128
	err = workflow.ExecuteActivity(ctx, MatchPresentment, accountId, amount, redisClient).Get(ctx, &transferId)
	if err != nil {
		log.Printf("Error in finding pending auth: %s", err)
		return false, err
	}

	if transferId == InvalidTransferId {
		log.Printf("No pending auth found for %d on account %s", amount, accountId)
		return false, nil
	}

	err = workflow.ExecuteActivity(ctx, PostPendingTransfer, transferId, accountId, amount, redisClient).Get(ctx, nil)
	if err != nil {
		log.Printf("Could not post pending transfer: %s", err)
		return false, err
	}

	log.Printf("Matched placement with presentment for %d on account %s with transfer %s", amount, accountId, transferId)
	return true, nil
}
