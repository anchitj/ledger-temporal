package app

import (
	"context"
	"encore.app/app/workflow"
	"encore.dev/rlog"
	"github.com/google/uuid"
	tbtypes "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
	"go.temporal.io/sdk/client"
)

type PresentResponse struct {
	PresentmentMatched bool
}

//encore:api public method=POST path=/present/:accountId/:amount
func (s *Service) Present(ctx context.Context, accountId string, amount uint64) (*PresentResponse, error) {
	accountIdCasted, _ := tbtypes.HexStringToUint128(accountId)
	options := client.StartWorkflowOptions{
		ID:        uuid.New().String(),
		TaskQueue: taskQueue,
	}
	we, err := s.temporalClient.ExecuteWorkflow(ctx, options, workflow.Present, accountIdCasted, amount)

	if err != nil {
		rlog.Error("failed to start workflow", "error", err)
		return &PresentResponse{PresentmentMatched: false}, err
	}
	rlog.Info("started workflow", "id", we.GetID(), "run_id", we.GetRunID())

	var presentmentMatched bool
	err = we.Get(ctx, &presentmentMatched)
	if err != nil {
		return &PresentResponse{PresentmentMatched: false}, err
	}
	return &PresentResponse{PresentmentMatched: presentmentMatched}, nil
}
