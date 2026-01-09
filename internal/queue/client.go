package queue

import "github.com/hibiken/asynq"

type Client struct {
	inner *asynq.Client
}

func NewClient(redisAddr, redisPass string, redisDB int) *Client {
	c := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       redisDB,
	})
	return &Client{inner: c}
}

func (c *Client) Close() error {
	return c.inner.Close()
}

func (c *Client) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return c.inner.Enqueue(task, opts...)
}
