package app

import (
	"context"
	"encore.app/app/workflow"
	encore "encore.dev"
	"fmt"
	"github.com/go-redis/redis"
	tb "github.com/tigerbeetledb/tigerbeetle-go"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"log"
)

const (
	LedgerId = 1
)

var (
	envName   = encore.Meta().Environment.Name
	taskQueue = envName + "-task-queue"
)

//encore:service
type Service struct {
	temporalClient client.Client
	temporalWorker worker.Worker
	redisClient    *redis.Client
	tbClient       tb.Client
}

func initService() (*Service, error) {
	c, err := client.Dial(client.Options{})
	if err != nil {
		return nil, fmt.Errorf("create temporal client: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	tbClient, err := tb.NewClient(0, []string{"3000"}, 1)
	if err != nil {
		log.Printf("Error creating tbclient: %s", err)
	}

	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflow(workflow.Auth)
	w.RegisterWorkflow(workflow.Present)
	w.RegisterWorkflow(workflow.Void)
	activities := &workflow.Activities{RedisClient: redisClient, TbClient: tbClient}
	w.RegisterActivity(activities)

	err = w.Start()
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("start temporal worker: %v", err)
	}
	return &Service{temporalClient: c, temporalWorker: w, redisClient: redisClient, tbClient: tbClient}, nil
}

func (s *Service) Shutdown(force context.Context) {
	s.temporalClient.Close()
	s.temporalWorker.Stop()
	s.tbClient.Close()
	err := s.redisClient.Close()
	if err != nil {
		log.Print("Error in closing redis client: ", err)
	}
}
