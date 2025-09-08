package amqp

import (
	"context"
	"fmt"
	"time"

	"CommandHandler/utils"
	streadway "github.com/streadway/amqp"
)

type Client struct {
	log  *utils.Logger
	conn *streadway.Connection
	ch   *streadway.Channel
}

func NewClient(log *utils.Logger) *Client { return &Client{log: log} }

// Connect dengan exponential backoff ringan
func (c *Client) Connect(ctx context.Context, url string) error {
	var (
		conn *streadway.Connection
		err  error
	)
	for attempt := 0; attempt < 8; attempt++ {
		conn, err = streadway.Dial(url)
		if err == nil {
			break
		}
		// backoff: 200ms, 400ms, 800ms, ...
		delay := time.Duration(1<<uint(attempt)) * 200 * time.Millisecond
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	c.conn = conn
	c.ch = ch
	return nil
}

func (c *Client) Close() {
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func (c *Client) DeclareExchange(ctx context.Context, name, kind string, durable bool) error {
	return c.ch.ExchangeDeclare(name, kind, durable, false, false, false, nil)
}

func (c *Client) DeclareQueue(ctx context.Context, name string) error {
	_, err := c.ch.QueueDeclare(name, true, false, false, false, nil)
	return err
}

func (c *Client) BindQueue(ctx context.Context, queue, exchange, routingKey string) error {
	return c.ch.QueueBind(queue, routingKey, exchange, false, nil)
}

func (c *Client) SetupRepairQueue(ctx context.Context, storeID string) (queueName, routingKey string, err error) {
	const (
		exchange = "REPAIR_TRANSACTION"
		queue    = "COMMAND_REPAIR"
		exchKind = "topic"
	)
	routingKey = fmt.Sprintf("STORE.%s.COMMAND", storeID)

	if err = c.DeclareExchange(ctx, exchange, exchKind, true); err != nil {
		return "", "", err
	}
	if err = c.DeclareQueue(ctx, queue); err != nil {
		return "", "", err
	}
	if err = c.BindQueue(ctx, queue, exchange, routingKey); err != nil {
		return "", "", err
	}
	return queue, routingKey, nil
}

func (c *Client) Channel() *streadway.Channel {
	return c.ch
}
