package middleware

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
)

func ServeFile(filepath string) http.HandlerFunc {
	data, err := os.ReadFile(filepath)
	if err != nil {
		panic(err)
	}
	return func(rw http.ResponseWriter, r *http.Request) {
		_, _ = rw.Write(data)
	}
}

func LogHTTP(f http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.Host, r.URL.Path)
		f(rw, r)
	}
}

func MustMethod(method string, f http.HandlerFunc) http.HandlerFunc {
	invalidMethodResponse := []byte("Method must be " + method)
	return func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			rw.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = rw.Write(invalidMethodResponse)
		} else {
			f(rw, r)
		}
	}
}

func SwitchMethod(switches map[string]http.HandlerFunc) http.HandlerFunc {
	invalidMethodResponseBuf := bytes.Buffer{}

	invalidMethodResponseBuf.WriteString("Invalid Method: Expected methods ")

	const separator = ", "
	for method := range switches {
		invalidMethodResponseBuf.WriteString(method)
		invalidMethodResponseBuf.WriteString(separator)
	}
	invalidMethodResponseBuf.Truncate(invalidMethodResponseBuf.Len() - len(separator)) // remove final separator

	invalidMethodResponseBuf.WriteString(" but instead got ")

	return func(rw http.ResponseWriter, r *http.Request) {
		if f, ok := switches[r.Method]; ok {
			f(rw, r)
		} else {
			rw.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = rw.Write(invalidMethodResponseBuf.Bytes())
			_, _ = io.WriteString(rw, r.Method)
		}
	}
}
