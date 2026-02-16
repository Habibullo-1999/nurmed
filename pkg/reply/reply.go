package reply

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/fx"

	"nurmed/pkg/logger"
)

var Module = fx.Invoke(New)

type Params struct {
	fx.In
	Logger logger.ILogger
}

var iLogger logger.ILogger

func New(params Params) {
	iLogger = params.Logger
}

func Json(w http.ResponseWriter, status int, data interface{}) {

	reply, err := json.Marshal(data)
	if err != nil {
		iLogger.Error(nil, fmt.Sprintf("Err on Json, json.Marshal(data) %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write(reply)
}
