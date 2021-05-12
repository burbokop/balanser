package httptools

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func WriteError(w http.ResponseWriter, status int, err error) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)

	jsonErr := struct {
		Error            int
		ErrorDescription string
	}{
		status,
		err.Error(),
	}

	log.Printf("status %d, error: %s", status, err)

	err = json.NewEncoder(w).Encode(jsonErr)
	return err
}

func WriteJSONResponseOrDie(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			WriteError(w, http.StatusInternalServerError, err)
			log.Fatal(err)
			return
		}
	}
}

func DecodeBodyAndClose(body io.ReadCloser, v interface{}) error {
	decoder := json.NewDecoder(body)
	defer body.Close()
	err := decoder.Decode(v)
	if err != nil {
		return fmt.Errorf("can not decode body: %s", err)
	}
	return nil
}
