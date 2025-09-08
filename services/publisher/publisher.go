package publisher

import (
	"encoding/json"
	"fmt"
	"time"

	"CommandHandler/utils"
	"github.com/streadway/amqp"
)

const (
	statusExchange   = "REPAIR_STATUS_TRANSACTION"
	statusKind       = "direct"
	statusRoutingKey = "REPAIR.STATUS.UPDATED" // pastikan SAMA dgn binding di RabbitMQ
)

type TicketStatus struct {
	TicketID  string `json:"ticketId"`
	SenderNIK string `json:"senderNik"`
	Status    string `json:"status"`
}

func PublishTicketStatus(log *utils.Logger, ch *amqp.Channel, ticketID, senderNIK, status string) error {
	if status != "COMPLETED" && status != "FAILED" {
		log.Fail("invalid status", "status", status)
		return fmt.Errorf("invalid status: %s", status)
	}

	// pastikan exchange ada
	if err := ch.ExchangeDeclare(statusExchange, statusKind, true, false, false, false, nil); err != nil {
		log.Fail("exchange declare failed", "err", err)
		return err
	}

	if err := ch.Confirm(false); err != nil {
		log.Warn("publisher confirms not supported", "err", err)
	}
	acks := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

	// detect NO_ROUTE
	returns := ch.NotifyReturn(make(chan amqp.Return, 1))

	body, _ := json.Marshal(TicketStatus{TicketID: ticketID, SenderNIK: senderNIK, Status: status})

	// publish dengan mandatory=true agar unroutable masuk ke NotifyReturn
	if err := ch.Publish(
		statusExchange,
		statusRoutingKey,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	); err != nil {
		log.Fail("💥 status publish failed", "ticket", ticketID, "err", err)
		return err
	}

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	select {
	case r := <-returns:

		log.Fail("⛔ UNROUTABLE",
			"ticket", ticketID,
			"rk", r.RoutingKey,
			"reason", r.ReplyText,
		)
		return fmt.Errorf("unroutable: %s", r.ReplyText)

	case c := <-acks:
		if c.Ack {
			log.OK("✅ status published",
				"exchange", statusExchange,
				"rk", statusRoutingKey,
				"ticket", ticketID,
				"status", status,
			)
			return nil
		}
		log.Fail("💥 publish NACKed", "ticket", ticketID)
		return fmt.Errorf("publish nacked")

	case <-timer.C:
		log.Warn("publish confirm timeout (assume routed)",
			"exchange", statusExchange,
			"rk", statusRoutingKey,
			"ticket", ticketID,
			"status", status,
		)
		return nil
	}
}
