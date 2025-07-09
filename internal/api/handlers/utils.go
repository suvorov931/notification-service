package handlers

import (
	"encoding/json"
	"net/http"
)

type respMessage struct {
	Message string `json:"message"`
	Id      int    `json:"id,omitempty"`
}

func writeResponseWithId(w http.ResponseWriter, id int, message string) error {
	w.Header().Set("Content-Type", "application/json")

	resp := respMessage{
		Message: message,
		Id:      id,
	}

	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		return err
	}

	return nil
}
