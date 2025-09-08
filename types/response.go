package types

type CommonResponse struct {
	TypeCommand CommandType `json:"typeCommand"`
	Handler     string      `json:"handler"`
	Status      string      `json:"status"` // "success" / "failed"
	Data        any         `json:"data"`
}

type ResponseRepairPayment struct {
	TipeBayar     string `json:"tipeBayar"`
	LogCashdrawer string `json:"logCashdrawer"`
}
