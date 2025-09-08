package consumer

import (
	"context"
	"encoding/json"
	"fmt" // <-- tambah

	"CommandHandler/services/dispatcher"
	"CommandHandler/types"
	"CommandHandler/utils"

	"github.com/streadway/amqp"
)

// unwrapToCommand:
// - Jika body = { "data": {...} } → unmarshal data ke Command
// - Jika tidak ada "data" → unmarshal body langsung ke Command
func unwrapToCommand(body []byte, out *types.Command) error {
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err == nil && len(env.Data) > 0 {
		return json.Unmarshal(env.Data, out)
	}
	return json.Unmarshal(body, out)
}

func Start(ctx context.Context, log *utils.Logger, ch *amqp.Channel, queue string, h *dispatcher.Handler) error {
	// Batasi in-flight messages agar stabil
	if err := ch.Qos(10, 0, false); err != nil {
		log.Fail("set QoS failed", "err", err)
	}

	const consumerTag = "command-repair-consumer"
	deliveries, err := ch.Consume(queue, consumerTag, false, false, false, false, nil)
	if err != nil {
		return err
	}
	log.OK("Consumer started", "queue", queue)

	// Biar bisa berhenti rapi saat ctx selesai
	go func() { <-ctx.Done(); _ = ch.Cancel(consumerTag, false) }()

	for {
		select {
		case <-ctx.Done():
			log.Warn("consumer context done")
			return ctx.Err()

		case d, ok := <-deliveries:
			if !ok {
				log.Warn("deliveries channel closed")
				return nil
			}

			func() {
				// Safety net: jangan sampai panic matiin consumer
				defer func() {
					if r := recover(); r != nil {
						log.Fail("panic in consumer", "recover", r)
						_ = d.Nack(false, false) // drop
					}
				}()

				var cmd types.Command
				if err := unwrapToCommand(d.Body, &cmd); err != nil {
					log.Fail("invalid message JSON", "err", err)
					_ = d.Nack(false, false) // drop
					return
				}

				log.Info("📥 received",
					"queue", queue,
					"type", cmd.CommandType,
					"ticket", cmd.TicketID,
					"idStore", cmd.IDStore,
				)

				resp, _ := h.Dispatch(ctx, cmd) // dispatcher handle status publish

				if resp.Status != "success" {
					// ambil pesan error yang ramah
					errText := ""
					switch v := resp.Data.(type) {
					case map[string]any:
						if e, ok := v["error"]; ok {
							errText = fmt.Sprint(e)
						} else {
							b, _ := json.Marshal(v)
							errText = string(b)
						}
					default:
						b, _ := json.Marshal(v)
						errText = string(b)
					}

					log.Fail("processed (failed)",
						"ticket", cmd.TicketID,
						"type", cmd.CommandType,
						"error", errText,
					)
				} else {
					log.OK("processed", "ticket", cmd.TicketID, "status", resp.Status, "handler", resp.Handler)
				}

				_ = d.Ack(false)
			}()
		}
	}
}
