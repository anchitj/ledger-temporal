package app

import (
	"context"
	"encore.app/app/workflow"
	"encore.dev/rlog"
	"github.com/google/uuid"
	tbtypes "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
	"go.temporal.io/sdk/client"
)

type AuthorizeResponse struct {
	Authorized bool
}

//encore:api public method=POST path=/authorize/:accountId/:amount
func (s *Service) Authorize(ctx context.Context, accountId string, amount uint64) (*AuthorizeResponse, error) {
	options := client.StartWorkflowOptions{
		ID:        uuid.New().String(),
		TaskQueue: taskQueue,
	}
	accountIdCasted, _ := tbtypes.HexStringToUint128(accountId)
	we, err := s.temporalClient.ExecuteWorkflow(ctx, options, workflow.Auth, accountIdCasted, amount, s.redisClient)

	if err != nil {
		rlog.Error("failed to start workflow", "error", err)
		return &AuthorizeResponse{Authorized: false}, err
	}
	rlog.Info("started workflow", "id", we.GetID(), "run_id", we.GetRunID())

	var authorized bool
	err = we.Get(ctx, &authorized)
	if err != nil {
		return &AuthorizeResponse{Authorized: false}, err
	}
	return &AuthorizeResponse{Authorized: authorized}, nil
}
