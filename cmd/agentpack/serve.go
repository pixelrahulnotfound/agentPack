// Live discovery HTTP API: `agentpack serve --registry DIR --port 8080`.
//
// Endpoints (all JSON, same envelope shape as --json CLI output):
//
//	GET /agents        list every indexed agent
//	GET /agents/:name  one registry entry (404 if unknown)
//	GET /search?q=...  ranked matches, same scoring as `search`
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"

	"agentpack/internal/output"
	"agentpack/internal/registry"
)

func cmdServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	reg := fs.String("registry", "registry-index", "registry index dir")
	port := fs.String("port", "8080", "listen port (or host:port)")
	_ = parseMixed(fs, args, map[string]bool{"registry": true, "port": true})

	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(v)
	}
	ok := func(w http.ResponseWriter, data any) {
		write(w, http.StatusOK, output.Envelope{Ok: true, Data: data})
	}
	fail := func(w http.ResponseWriter, status int, code, msg string) {
		write(w, status, output.Envelope{Ok: false, Error: &output.ErrBody{Code: code, Message: msg}})
	}

	mux.HandleFunc("/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			fail(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
			return
		}
		entries, err := registry.List(*reg)
		if err != nil {
			fail(w, http.StatusInternalServerError, "registry_error", err.Error())
			return
		}
		ok(w, entries)
	})
	mux.HandleFunc("/agents/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			fail(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/agents/")
		if name == "" || strings.Contains(name, "/") {
			fail(w, http.StatusNotFound, "not_found", "unknown agent")
			return
		}
		e, err := registry.Read(*reg, name)
		if err != nil {
			fail(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		ok(w, e)
	})
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			fail(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
			return
		}
		matches, err := registry.Search(*reg, r.URL.Query().Get("q"))
		if err != nil {
			fail(w, http.StatusInternalServerError, "registry_error", err.Error())
			return
		}
		ok(w, matches)
	})

	addr := *port
	if !strings.Contains(addr, ":") {
		addr = "127.0.0.1:" + addr
	}
	fmt.Printf("agentpack discovery API on http://%s (registry: %s)\n", addr, *reg)
	if err := http.ListenAndServe(addr, mux); err != nil {
		output.Failure("serve_failed", err.Error(), "")
		return err
	}
	return nil
}
