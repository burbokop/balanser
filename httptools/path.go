package httptools

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func GetStringFromPath(paramName string, r *http.Request) (string, error) {
	vars := mux.Vars(r)
	if vars[paramName] == "" {
		return "", fmt.Errorf("%s can not be empty", paramName)
	}
	return vars[paramName], nil
}
