package cluster

import "net/http"

func codeToMsg(code int) *Msg {
	return &Msg{
		Code:    http.StatusBadRequest,
		Message: http.StatusText(http.StatusBadGateway),
	}
}
