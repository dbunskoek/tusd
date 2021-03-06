package tusd_test

import (
	"net/http"
	"testing"

	. "github.com/tus/tusd"
)

func TestOptions(t *testing.T) {
	handler, _ := NewHandler(Config{
		MaxSize: 400,
	})

	(&httpTest{
		Name:   "Successful request",
		Method: "OPTIONS",
		Code:   http.StatusNoContent,
		ResHeader: map[string]string{
			"Tus-Extension": "creation",
			"Tus-Version":   "1.0.0",
			"Tus-Resumable": "1.0.0",
			"Tus-Max-Size":  "400",
		},
	}).Run(handler, t)

	(&httpTest{
		Name:   "Invalid or unsupported version",
		Method: "POST",
		ReqHeader: map[string]string{
			"Tus-Resumable": "foo",
		},
		Code: http.StatusPreconditionFailed,
	}).Run(handler, t)
}
