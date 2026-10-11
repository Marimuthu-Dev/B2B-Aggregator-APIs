package dto

type StateResponse struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type CityResponse struct {
	ID      int32  `json:"id"`
	Name    string `json:"name"`
	StateID int32  `json:"stateId"`
}

type StateIDPathParam struct {
	StateID int32 `uri:"state_id" binding:"required"`
}
