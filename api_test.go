package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestSplitAPICommand(t *testing.T) {
	for _, test := range []struct {
		command string
		want    []string
	}{
		{`dotnet MyApi.dll --urls http://127.0.0.1:1111`, []string{"dotnet", "MyApi.dll", "--urls", "http://127.0.0.1:1111"}},
		{`"C:\Program Files\dotnet\dotnet.exe" 'My API.dll' ""`, []string{`C:\Program Files\dotnet\dotnet.exe`, "My API.dll", ""}},
		{`api --name="hello world" 'Unicode λ' 'say "hi"'`, []string{"api", "--name=hello world", "Unicode λ", `say "hi"`}},
		{`api $HOME *.dll | next`, []string{"api", "$HOME", "*.dll", "|", "next"}},
	} {
		got, err := splitAPICommand(test.command)
		if err != nil || !reflect.DeepEqual(got, test.want) {
			t.Fatalf("split %q = %#v, %v; want %#v", test.command, got, err, test.want)
		}
	}
	for _, command := range []string{"", "  ", `""`, `api "unterminated`} {
		if _, err := splitAPICommand(command); err == nil {
			t.Fatalf("accepted invalid command %q", command)
		}
	}
}

func TestAPIOptionsPrecedence(t *testing.T) {
	config, cleanup := writeTestConfig(t, "API_COMMAND=dotnet Config.dll\nAPI_CWD=./config cwd\nAPI_FORWARD_TO=http://localhost:1111\nAPI_PREFIX=/config\n")
	defer cleanup()
	for _, source := range []string{"config", "environment", "CLI"} {
		flags := flag.NewFlagSet("test", flag.ContinueOnError)
		var options apiOptions
		registerAPIFlags(flags, &options)
		lookup := func(key string) (string, bool) {
			if source == "config" {
				return "", false
			}
			values := map[string]string{
				"GOSTATIC_API_COMMAND": "dotnet Env.dll", "GOSTATIC_API_CWD": "./env cwd",
				"GOSTATIC_API_FORWARD_TO": "http://localhost:2222", "GOSTATIC_API_PREFIX": "/env",
			}
			value, ok := values[key]
			return value, ok
		}
		var args []string
		want := apiOptions{"dotnet Config.dll", "./config cwd", "http://localhost:1111", "/config"}
		if source != "config" {
			want = apiOptions{"dotnet Env.dll", "./env cwd", "http://localhost:2222", "/env"}
		}
		if source == "CLI" {
			args = []string{"--api-command=dotnet CLI.dll", "--api-cwd=./CLI cwd", "--api-forward-to=", "--api-prefix=/"}
			want = apiOptions{"dotnet CLI.dll", "./CLI cwd", "", "/"}
		}
		if err := loadOptions(config, flags, lookup, args); err != nil || options != want {
			t.Fatalf("%s: got %+v, %v; want %+v", source, options, err, want)
		}
	}
}

func TestAPIProxyRouting(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		w.Header().Set("X-Backend", "yes")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "%s %s %s host=%s", r.Method, r.RequestURI, body, r.Header.Get("X-Forwarded-Host"))
	}))
	defer backend.Close()
	static := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "static") })
	for _, test := range []struct {
		prefix, path string
		proxy        bool
	}{
		{"/api", "/api", true}, {"/api", "/api/customers?x=a%20b", true},
		{"/api/", "/api/customers", true}, {"/api", "/apiculture", false},
		{"/api", "/", false}, {"/", "/customers", true},
	} {
		handler, _, err := apiHandler(static, apiOptions{forwardTo: backend.URL, prefix: test.prefix})
		if err != nil {
			t.Fatal(err)
		}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("POST", "http://preview.test"+test.path, strings.NewReader("payload"))
		request.Header.Set("X-Forwarded-Host", "spoofed")
		handler.ServeHTTP(recorder, request)
		if test.proxy {
			want := "POST " + test.path + " payload host=preview.test"
			if recorder.Code != http.StatusCreated || recorder.Body.String() != want || recorder.Header().Get("X-Backend") != "yes" {
				t.Fatalf("proxy %q: status=%d body=%q headers=%v", test.path, recorder.Code, recorder.Body.String(), recorder.Header())
			}
		} else if recorder.Body.String() != "static" {
			t.Fatalf("static route %q returned %q", test.path, recorder.Body.String())
		}
	}
}

func TestAPIHandlerValidation(t *testing.T) {
	for _, options := range []apiOptions{
		{cwd: "orphan"}, {forwardTo: "relative", prefix: "/api"},
		{forwardTo: "ftp://localhost", prefix: "/api"},
		{forwardTo: "http://localhost:0", prefix: "/api"},
		{forwardTo: "http://localhost:70000", prefix: "/api"},
		{forwardTo: "http://localhost", prefix: "api"},
		{forwardTo: "http://user:password@localhost", prefix: "/api"},
		{forwardTo: "http://localhost?query=x", prefix: "/api"},
	} {
		if _, _, err := apiHandler(http.NotFoundHandler(), options); err == nil {
			t.Fatalf("accepted %+v", options)
		}
	}
}

func TestUnavailableAPIProducesBadGateway(t *testing.T) {
	backend := httptest.NewServer(http.NotFoundHandler())
	backend.Close()
	handler, _, err := apiHandler(http.NotFoundHandler(), apiOptions{forwardTo: backend.URL, prefix: "/api"})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/api", nil))
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("got %d; want 502", recorder.Code)
	}
}
