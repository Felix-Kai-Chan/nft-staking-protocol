package mq

import (
	"context"
	"encoding/json"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"

	"staking-offchain/internal/dao"
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
	db      *gorm.DB
}

func NewConsumer(url, queue string, db *gorm.DB) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	_, err = ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		conn:    conn,
		channel: ch,
		queue:   queue,
		db:      db,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		c.queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	slog.Info("consumer started", "queue", c.queue)

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			if err := c.handleMessage(ctx, msg); err != nil {
				slog.Error("failed to handle message", "error", err)
				msg.Nack(false, true)
			} else {
				msg.Ack(false)
			}
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	var event EventMessage
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		return err
	}

	slog.Info("consumed event", "tx", event.TxHash, "sig", event.EventSig)

	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		eventLogDAO := dao.NewEventLogDAO(tx)
		if err := eventLogDAO.Insert(ctx, event.TxHash, event.LogIndex, event.BlockNumber); err != nil {
			if err == dao.ErrDuplicate {
				slog.Info("duplicate event skipped", "tx", event.TxHash)
				return nil
			}
			return err
		}

		switch event.EventSig {
		case "Staked":
			stakeDAO := dao.NewStakeDAO(tx)
			return stakeDAO.SaveStake(ctx, &dao.Stake{
				UserAddress: event.User,
				Amount:      event.Amount,
				StakedAt:    event.BlockNumber,
				LastUpdated: event.BlockNumber,
			})
		case "Claimed":
			claimDAO := dao.NewClaimDAO(tx)
			return claimDAO.SaveClaim(ctx, event.User, event.Amount)
		}
		return nil
	})
}

func (c *Consumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
