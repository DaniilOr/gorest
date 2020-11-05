package remux

import (
	"context"
	"errors"
	"github.com/DaniilOr/gorest/pkg/middleware"
	"net/http"
	"regexp"
	"strings"
	"sync"
)
type Method string
type ReMUX struct{
	mu				 sync.RWMutex
	plain 			map[Method]map[string] http.Handler
	regex 			map[Method]map[*regexp.Regexp]http.Handler
	notFoundHandler http.Handler
}

type contextKey struct {
	name string
}

type Params struct {
	Named      map[string]string
	Positional []string
}
const (
	GET     Method = "GET"
	POST    Method = "POST"
	PUT     Method = "PUT"
	PATCH   Method = "PATCH"
	DELETE  Method = "DELETE"
	OPTIONS Method = "OPTIONS"
	HEAD    Method = "HEAD"
)
var paramsContextKey = &contextKey{"ReMUX context"}
var defaultNotFound = func(w http.ResponseWriter, r *http.Request){
	w.WriteHeader(http.StatusNotFound)
}
var (
	ErrInvalidPath      = errors.New("invalid path")
	ErrInvalidMethod    = errors.New("invalid http method")
	ErrNilHandler       = errors.New("nil handler")
	ErrAmbiguousMapping = errors.New("ambiguous mapping")
	ErrNoParams         = errors.New("no params")
)
func (c*contextKey) ToString() string{
	return c.name
}
func CreateNewReMUX()*ReMUX{
	return &ReMUX{
		notFoundHandler:http.HandlerFunc(defaultNotFound),
	}
}
func (r*ReMUX) NewPlain(method Method, path string, handler http.Handler, middlewares ...middleware.Middleware) error{
	if !isValidMethod(method) {
		return ErrInvalidMethod
	}

	if !strings.HasPrefix(path, "/") {
		return ErrInvalidPath
	}

	if handler == nil {
		return ErrNilHandler
	}
	handler = wrapHandler(handler, middlewares...)
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.plain[method][path]
	if ok{
		return ErrAmbiguousMapping
	}
	if r.plain == nil {
		r.plain = make(map[Method]map[string]http.Handler)
	}
	if r.plain[method] == nil {
		r.plain[method] = make(map[string]http.Handler)
	}
	r.plain[method][path] = handler
	return nil
}
func (r*ReMUX) NewRegex(method Method, handler http.Handler, path *regexp.Regexp, 	middlewares ...middleware.Middleware) error{
	if !isValidMethod(method){
		return ErrInvalidMethod
	}
	if handler == nil{
		return ErrNilHandler
	}
	if !strings.HasPrefix(path.String(), `^/`) {
		return ErrInvalidPath
	}
	if !strings.HasSuffix(path.String(), `$`){
		return ErrInvalidPath
	}
	handler = wrapHandler(handler, middlewares...)
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.regex[method][path]
	if ok{
		return ErrAmbiguousMapping
	}
	if r.regex == nil{
		r.regex = make(map[Method]map[*regexp.Regexp]http.Handler)
	}
	if r.regex[method] == nil {
		r.regex[method] = make(map[*regexp.Regexp]http.Handler)
	}

	r.regex[method][path] = handler
	return nil
}
func (r *ReMUX) SetNotFoundHandler(handler http.Handler) error {
	if handler == nil {
		return ErrNilHandler
	}
	r.mu.Lock()
	r.notFoundHandler = handler
	r.mu.Unlock()
	return nil
}
func (r*ReMUX) ServeHTTP(w http.ResponseWriter, req *http.Request){
	r.mu.RLock()
	var resultHandler http.Handler
	if handlers, exist := r.plain[Method(req.Method)]; exist{
		if handler, ok := handlers[req.URL.Path]; ok{
			resultHandler = handler
		}
	}

	if resultHandler == nil{
		if handlers, exist := r.regex[Method(req.Method)]; exist{
			for path, handler := range handlers{
				if matches := path.FindStringSubmatch(req.URL.Path); matches != nil{
					params := &Params{
						make(map[string]string),
						matches[1:], // FindStringSubmatch в 0 индексе хранит всю строку
					}
					for index, name := range path.SubexpNames(){
						if name == ""{
							continue
						}
						params.Named[name] = matches[index]
					}
					ctx := context.WithValue(req.Context(), paramsContextKey, params)
					req = req.WithContext(ctx)
					resultHandler = handler
					break
				}
			}
		}
	}
	if resultHandler == nil {
		resultHandler = r.notFoundHandler
	}

	r.mu.RUnlock()
	resultHandler.ServeHTTP(w, req)
}

func isValidMethod(method Method) bool{
	allowedMethods := map[Method]bool{GET:true, POST:true, PUT:true, PATCH:true, DELETE:true, OPTIONS:true, HEAD:true}
	_, ok := allowedMethods[method]
	return ok
}

func wrapHandler(handler http.Handler, middlewares ...middleware.Middleware) http.Handler {
	for _, m := range middlewares {
		handler = m(handler)
	}
	return handler
}

func PathParams(ctx context.Context) (*Params, error) {
	params, ok := ctx.Value(paramsContextKey).(*Params)
	if !ok {
		return nil, ErrNoParams
	}
	return params, nil
}