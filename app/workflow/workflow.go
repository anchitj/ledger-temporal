package workflow

import (
	tbtypes "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
	"go.temporal.io/sdk/workflow"
	"log"
	"time"
)

const (
	CreditAccountId           = "1234567"
	AuthorizationHoldDuration = 10 * time.Second
	InvalidTransferIdString   = "00000000000000000000000000000000"
)

var InvalidTransferId, _ = tbtypes.HexStringToUint128(InvalidTransferIdString)

func Auth(ctx workflow.Context, accountId tbtypes.Uint128, amount uint64) (tbtypes.Uint128, error) {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
	}

	ctx = workflow.WithActivityOptions(ctx, options)

	var a *Activities

	var accountExistsWithSufficientBalance bool
	err := workflow.ExecuteActivity(ctx, a.CheckAccountExistsWithSufficientBalance, accountId, amount).Get(ctx, &accountExistsWithSufficientBalance)
	if err != nil {
		log.Printf("Could not check account existence: %s", err)
		return InvalidTransferId, err
	}
	if !accountExistsWithSufficientBalance {
		log.Printf("Account %s does not exist or doesn't have sufficient balance", accountId)
		return InvalidTransferId, nil
	}

	creditAccountIdCasted, _ := tbtypes.HexStringToUint128(CreditAccountId)
	log.Printf("starting authorization for %d on account %s %s %s", amount, accountId, accountId.String(), creditAccountIdCasted.String())

	var transferId tbtypes.Uint128
	err = workflow.ExecuteActivity(ctx, a.PlaceAuthorization, accountId, creditAccountIdCasted, amount).Get(ctx, &transferId)
	if err != nil {
		log.Printf("Could not place authoriation: %s", err)
		return InvalidTransferId, err
	}

	//cwo := workflow.ChildWorkflowOptions{
	//	WorkflowID:        uuid.New().String(),
	//	ParentClosePolicy: enums.ParentClosePolicy(3),
	//}
	//ctx = workflow.WithChildOptions(ctx, cwo)
	//workflow.ExecuteChildWorkflow(ctx, Void, transferId, AuthorizationHoldDuration)
	//err = workflow.ExecuteChildWorkflow(ctx, Void, transferId, AuthorizationHoldDuration).GetChildWorkflowExecution().Get(ctx, nil)
	//if err != nil {
	//	log.Printf("Could not void pending transfer: %s", err)
	//	return false, err
	//}

	log.Printf("Placed authorization for %d on account %s", amount, accountId)

	return transferId, nil
}

func Present(ctx workflow.Context, accountId tbtypes.Uint128, amount uint64) (bool, error) {
	//retrypolicy := &temporal.RetryPolicy{
	//	MaximumAttempts: 1,
	//}
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
		//RetryPolicy:         retrypolicy,
	}

	ctx = workflow.WithActivityOptions(ctx, options)
	var a *Activities

	var accountExists bool
	err := workflow.ExecuteActivity(ctx, a.CheckAccountExists, accountId).Get(ctx, &accountExists)
	if err != nil {
		log.Printf("Could not check account existence: %s", err)
		return false, err
	}
	if !accountExists {
		log.Printf("Account %s does not exist", accountId)
		return false, nil
	}

	var transferId tbtypes.Uint128
	err = workflow.ExecuteActivity(ctx, a.MatchPresentment, accountId, amount).Get(ctx, &transferId)
	if err != nil {
		log.Printf("Error in finding pending auth: %s", err)
		return false, err
	}

	if transferId == InvalidTransferId {
		log.Printf("No pending auth found for %d on account %s", amount, accountId)
		return false, nil
	}

	err = workflow.ExecuteActivity(ctx, a.PostPendingTransfer, transferId, accountId, amount).Get(ctx, nil)
	if err != nil {
		log.Printf("Could not post pending transfer: %s", err)
		return false, err
	}

	log.Printf("Matched placement with presentment for %d on account %s with transfer %s", amount, accountId, transferId)
	return true, nil
}

func Void(ctx workflow.Context, transferId tbtypes.Uint128) error {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
	}
	ctx = workflow.WithActivityOptions(ctx, options)

	err := workflow.Sleep(ctx, AuthorizationHoldDuration)
	if err != nil {
		return err
	}

	var a *Activities

	var isPendingTransfer bool
	err = workflow.ExecuteActivity(ctx, a.IsPendingTransfer, transferId).Get(ctx, &isPendingTransfer)
	if err != nil {
		log.Printf("Could not check pending transfer: %s", err)
		return err
	}
	if !isPendingTransfer {
		log.Printf("Transfer %s is not pending. Exiting", transferId)
		return nil
	}

	err = workflow.ExecuteActivity(ctx, a.VoidAuthorization, transferId).Get(ctx, nil)
	if err != nil {
		log.Printf("Could not void pending transfer: %s", err)
		return err
	}

	log.Printf("Voided pending transfer %s", transferId)
	return nil
}
