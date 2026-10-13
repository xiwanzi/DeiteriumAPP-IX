package httpapi

import (
	"net/http"
	"time"
)

func v2Success(w http.ResponseWriter, r *http.Request, data any) {
	writeJSON(w, http.StatusOK, map[string]any{"requestId": requestID(r), "data": data, "serverTime": time.Now().UTC()})
}

func v2List(w http.ResponseWriter, r *http.Request, data any, nextCursor any, hasMore bool) {
	writeJSON(w, http.StatusOK, map[string]any{"requestId": requestID(r), "data": data, "serverTime": time.Now().UTC(),
		"page": map[string]any{"nextCursor": nextCursor, "hasMore": hasMore}})
}
