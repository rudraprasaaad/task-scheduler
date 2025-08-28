package client

import (
	"context"
	"fmt"
	"io"
	"log"

	taskpb "github.com/rudraprasaaad/task-scheduler/internal/grpc/generated/task"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TaskClient struct {
	conn   *grpc.ClientConn
	client taskpb.TaskServiceClient
}

func NewTaskClient(serverAddr string) (*TaskClient, error) {
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		return nil, err
	}

	client := taskpb.NewTaskServiceClient(conn)

	return &TaskClient{
		conn:   conn,
		client: client,
	}, nil
}

func (c *TaskClient) Close() error {
	return c.conn.Close()
}

func (c *TaskClient) GetAvailableTasks(ctx context.Context, workerID string, limit int32) ([]*taskpb.Task, error) {
	req := &taskpb.GetTaskRequest{
		WorkerId: workerID,
		Limit:    limit,
	}

	res, err := c.client.GetAvailableTask(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.Tasks, nil
}

func (c *TaskClient) UpdateTask(ctx context.Context, task *taskpb.Task) error {
	req := &taskpb.UpdateTaskRequest{
		Task: task,
	}

	res, err := c.client.UpdateTask(ctx, req)
	if err != nil {
		return err
	}

	if res.Success {
		return fmt.Errorf("update failedL %s", res.Message)
	}

	return nil
}

func (c *TaskClient) StreamTasks(ctx context.Context, workerID string, limit int32, taskChan chan<- *taskpb.Task) error {
	req := &taskpb.GetTaskRequest{
		WorkerId: workerID,
		Limit:    limit,
	}

	stream, err := c.client.StreamTasks(ctx, req)
	if err != nil {
		return err
	}

	for {
		task, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Printf("Stream error: %v", err)
			return err
		}

		select {
		case taskChan <- task:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
