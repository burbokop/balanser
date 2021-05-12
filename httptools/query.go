package httptools

import (
	"fmt"
	"net/http"
)

func GetStringFromQuery(paramName string, mandatory bool, r *http.Request) (string, error) {
	params := r.URL.Query()
	if len(params[paramName]) == 0 || params[paramName][0] == "" {
		if mandatory {
			return "", fmt.Errorf("%s cannot be empty", paramName)
		}
		return "", nil
	}
	return params[paramName][0], nil
}
