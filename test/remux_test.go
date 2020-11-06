package test

import (
	"bytes"
	"github.com/DaniilOr/gorest/pkg/middleware/logger"
	"github.com/DaniilOr/gorest/pkg/middleware/recoverer"
	"github.com/DaniilOr/gorest/pkg/remux"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestReMUX_NewPlain(t *testing.T) {
	mux := remux.CreateNewReMUX()
	loggerMd := logger.Logger
	if err := mux.NewPlain(remux.GET, "/get", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(remux.GET))
	}), loggerMd); err != nil {
		t.Fatal(err)
	}
	if err := mux.NewPlain(remux.POST, "/post", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(remux.POST))
	})); err != nil {
		t.Fatal(err)
	}
	if err := mux.NewPlain(remux.PUT, "/put", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(remux.PUT))
	})); err != nil {
		t.Fatal(err)
	}
	type args struct {
		method remux.Method
		path   string
	}

	tests := []struct {
		name string
		args args
		want []byte
	}{
		{name: "GET", args: args{method: remux.GET, path: "/get"}, want: []byte(remux.GET)},
		{name: "POST", args: args{method: remux.POST, path: "/post"}, want: []byte(remux.POST)},
		{name:"PUT", args: args{method: remux.PUT, path: "/put"}, want: []byte(remux.PUT)},
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
	mux := remux.CreateNewReMUX()
	recoverer := recoverer.Recoverer
	type args struct {
		method remux.Method
		path   string
	}
	if err := mux.NewPlain(remux.PUT, "/put", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(remux.PUT))
	}), recoverer); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "GET", args: args{method: remux.GET, path: "/get"}, want: http.StatusNotFound},
		{name: "POST", args: args{method: remux.POST, path: "/post"}, want: http.StatusNotFound},
		{name: "PUT", args: args{method: remux.PUT, path: "/put/poi"}, want: http.StatusInternalServerError},
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
	mux := remux.CreateNewReMUX()
	recoverer := recoverer.Recoverer
	type args struct {
		method remux.Method
		path   string
	}
	if err := mux.NewPlain(remux.PUT, "/put", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		panic("panic!")
	}), recoverer); err != nil {
		t.Fatal(err)
	}
	if err := mux.NewPlain(remux.GET, "/get", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(remux.GET))
		panic("panic!")
	}), recoverer); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "PUT", args: args{method: remux.PUT, path: "/put"}, want: http.StatusInternalServerError},
		{name: "GET", args: args{method: remux.GET, path: "/get"}, want: http.StatusOK},
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
func TestReMux_Regex(t *testing.T) {
	mux := remux.CreateNewReMUX()
	getRegex, err := regexp.Compile(`^/resources/(?P<resourceId>\d+)/subresources/(?P<subresourceId>\d+)$`)
	if err != nil {
		t.Fatal(err)
	}
	if err := mux.NewRegex(remux.GET, http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			params, err := remux.PathParams(request.Context())
			if err != nil {
				t.Error(err)
			}
			writer.Write([]byte(params.Named["resourceId"]))
		},
	), getRegex); err != nil {
		t.Fatal(err)
	}
	postRegex, err := regexp.Compile(`^/resources/(?P<resourceId>\d+)/subresources/(?P<subresourceId>\d+)$`)
	if err != nil {
		t.Fatal(err)
	}
	if err := mux.NewRegex(remux.POST, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		params, err := remux.PathParams(request.Context())
		if err != nil {
			t.Error(err)
		}
		writer.Write([]byte(params.Named["subresourceId"]))
	}),postRegex, ); err != nil {
		t.Fatal(err)
	}
	if err := mux.NewRegex(remux.PUT, http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			params, err := remux.PathParams(request.Context())
			if err != nil {
				t.Error(err)
			}
			writer.Write([]byte(params.Named["resourceId"]))
		},
	), getRegex); err != nil {
		t.Fatal(err)
	}
	type args struct {
		method remux.Method
		path   string
	}

	tests := []struct {
		name string
		args args
		want []byte
	}{
		{name: "GET", args: args{method: remux.GET, path: "/resources/1/subresources/2"}, want: []byte("1")},
		{name: "POST", args: args{method: remux.POST, path: "/resources/1/subresources/2"}, want: []byte("2")},
		{name: "PUT", args: args{method: remux.PUT, path: "/resources/1/subresources/2"}, want: []byte("1")},
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
