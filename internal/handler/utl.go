package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

var (
	errStreamingUnsupported = errors.New("streaming unsupported")
)

func respond(w http.ResponseWriter, v interface{}, statusCode int) {
	b, err := json.Marshal(v)

	if err != nil {
		respondError(w, fmt.Errorf("could not marshal respond %v", err))
		return
	}

	w.Header().Set("content-type", "application/json ; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(b)
}

func respondError(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func writeSSE(w io.Writer, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("could not marshal response: %v\n", err)
		fmt.Fprintf(w, "error: %v\n\n", err)
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", b)

}
