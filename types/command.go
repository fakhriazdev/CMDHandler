package types

type CommandType string

const (
	CommandRepairPayment CommandType = "REPAIR_PAYMENT"
	CommandDeletePayment CommandType = "DELETE_PAYMENT"
)

type Command struct {
	IDStore     string      `json:"idStore"`
	TicketID    string      `json:"ticketId"`
	CommandType CommandType `json:"commandType"`
	Payload     any         `json:"payload"`
}
