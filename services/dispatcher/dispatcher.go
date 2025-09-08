package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	services "CommandHandler/services"
	"CommandHandler/services/publisher"
	"CommandHandler/types"
	"CommandHandler/utils"

	"github.com/streadway/amqp"
)

type Handler struct {
	Log *utils.Logger
	Ch  *amqp.Channel
	Svc *services.Service
}

func New(log *utils.Logger, ch *amqp.Channel, svc *services.Service) *Handler {
	return &Handler{Log: log, Ch: ch, Svc: svc}
}

func (h *Handler) Dispatch(ctx context.Context, cmd types.Command) (types.CommonResponse, error) {
	switch cmd.CommandType {

	case types.CommandRepairPayment:
		// Parse payload generic → struct yang benar
		var p types.PayloadRepairPayment
		b, _ := json.Marshal(cmd.Payload)
		if err := json.Unmarshal(b, &p); err != nil {
			_ = publisher.PublishTicketStatus(h.Log, h.Ch, cmd.TicketID, "", "FAILED")
			return types.CommonResponse{
				TypeCommand: types.CommandRepairPayment,
				Handler:     "TransactionService",
				Status:      "failed",
				Data:        map[string]any{"error": "invalid payload"},
			}, nil
		}

		// NIK wajib
		if strings.TrimSpace(p.SenderNIK) == "" {
			_ = publisher.PublishTicketStatus(h.Log, h.Ch, cmd.TicketID, "", "FAILED")
			return types.CommonResponse{
				TypeCommand: types.CommandRepairPayment,
				Handler:     "TransactionService",
				Status:      "failed",
				Data:        map[string]any{"error": "senderNik is required"},
			}, nil
		}

		// Jalankan service
		res, err := h.Svc.RepairPaymentMethod(ctx, p)
		if err != nil {
			_ = publisher.PublishTicketStatus(h.Log, h.Ch, cmd.TicketID, p.SenderNIK, "FAILED")
			return types.CommonResponse{
				TypeCommand: types.CommandRepairPayment,
				Handler:     "TransactionService",
				Status:      "failed",
				Data:        map[string]any{"error": err.Error()},
			}, nil
		}

		// Sukses
		_ = publisher.PublishTicketStatus(h.Log, h.Ch, cmd.TicketID, p.SenderNIK, "COMPLETED")
		return types.CommonResponse{
			TypeCommand: types.CommandRepairPayment,
			Handler:     "TransactionService",
			Status:      "success",
			Data:        res,
		}, nil

	default:
		// Command tidak dikenal → mark failed, dan kembalikan error supaya terlihat sebagai kesalahan konfigurasi
		_ = publisher.PublishTicketStatus(h.Log, h.Ch, cmd.TicketID, "", "FAILED")
		return types.CommonResponse{
			TypeCommand: cmd.CommandType,
			Handler:     "UnknownHandler",
			Status:      "failed",
			Data:        map[string]any{"error": "unsupported command type: " + string(cmd.CommandType)},
		}, errors.New("unsupported command type")
	}
}
