package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Json map[string]interface{}

type JsonResponse struct {
	text   string
	json   Json
	writer http.ResponseWriter
}

func (j Json) String() (s string) {
	b, err := json.Marshal(j)
	if err != nil {
		s = ""
		return
	}

	s = string(b)
	return
}

func NewJsonResponse(w http.ResponseWriter) *JsonResponse {
	j := &JsonResponse{
		writer: w,
	}

	j.writer.Header().Set("Content-Type", "application/json")
	return j
}

func (j *JsonResponse) Write(json Json) {
	j.json = json
	j.text = fmt.Sprint(j.json)

	fmt.Fprint(j.writer, j.text)
	return
}
