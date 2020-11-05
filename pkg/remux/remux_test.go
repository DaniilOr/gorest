package remux

import (
	"bytes"
	"github.com/DaniilOr/gorest/pkg/middleware/logger"
	"github.com/DaniilOr/gorest/pkg/middleware/recoverer"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReMUX_NewPlain(t *testing.T) {
	mux := CreateNewReMUX()
	loggerMd := logger.Logger
	if err := mux.NewPlain(GET, "/get", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(GET))
	}), loggerMd); err != nil {
		t.Fatal(err)
	}
	if err := mux.NewPlain(POST, "/post", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(POST))
	})); err != nil {
		t.Fatal(err)
	}
	if err := mux.NewPlain(PUT, "/put", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(PUT))
	})); err != nil {
		t.Fatal(err)
	}
	type args struct {
		method Method
		path   string
	}

	tests := []struct {
		name string
		args args
		want []byte
	}{
		{name: "GET", args: args{method: GET, path: "/get"}, want: []byte(GET)},
		{name: "POST", args: args{method: POST, path: "/post"}, want: []byte(POST)},
		{name:"PUT", args: args{method: PUT, path: "/put"}, want: []byte(PUT)},
	}

	for _, tt := range tests {
		request := httptest.NewRequest(string(tt.args.method), tt.args.path, nil)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		got := response.Body.Bytes()
		if !bytes.Equal(tt.want, got) {
			t.Errorf("got %s, want %s", got, tt.want)
		}
	}
}

func TestReMUX_SetNotFoundHandler(t *testing.T) {
	mux := CreateNewReMUX()
	recoverer := recoverer.Recoverer
	type args struct {
		method Method
		path   string
	}
	if err := mux.NewPlain(PUT, "/put", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(PUT))
	}), recoverer); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "GET", args: args{method: GET, path: "/get"}, want: http.StatusNotFound},
		{name: "POST", args: args{method: POST, path: "/post"}, want: http.StatusNotFound},
		{name: "PUT", args: args{method: PUT, path: "/put/poi"}, want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		request := httptest.NewRequest(string(tt.args.method), tt.args.path, nil)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		got := response.Code
		if tt.want != got {
			t.Errorf("got %v, want %v", got, tt.want)
		}
	}
}
func TestReMUX_Panic(t *testing.T) {
	mux := CreateNewReMUX()
	recoverer := recoverer.Recoverer
	type args struct {
		method Method
		path   string
	}
	if err := mux.NewPlain(PUT, "/put", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		panic("panic!")
	}), recoverer); err != nil {
		t.Fatal(err)
	}
	if err := mux.NewPlain(GET, "/get", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(GET))
		panic("panic!")
	}), recoverer); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "PUT", args: args{method: PUT, path: "/put"}, want: http.StatusInternalServerError},
		{name: "GET", args: args{method: GET, path: "/get"}, want: http.StatusOK},
	}

	for _, tt := range tests {
		request := httptest.NewRequest(string(tt.args.method), tt.args.path, nil)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		got := response.Code
		if tt.want != got {
			t.Errorf("got %v, want %v", got, tt.want)
		}
	}
}