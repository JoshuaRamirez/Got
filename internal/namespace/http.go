package namespace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/joshuaramirez/got/internal/identity"
)

// This file provides a network-transparent Store: HTTPStore is a Store client
// that talks to an HTTP server, and NewHTTPHandler wraps any Store as that
// server. It exists because the Store interface takes context.Context on every
// method precisely to allow a remote/persistent backing (see the interface
// doc); this is the concrete demonstration that a Store works across a network
// boundary, with the caller's ctx threaded onto each request.

// bindReq is the JSON body of a bind request.
type bindReq struct {
	Kind string `json:"kind"` // "ref" | "alias" | "projection"
	Name string `json:"name"`
	ID   string `json:"id"` // hex-encoded vertex id
}

// resolveResp is the JSON body of a resolve response.
type resolveResp struct {
	Found bool   `json:"found"`
	ID    string `json:"id,omitempty"` // hex-encoded vertex id when Found
}

// NewHTTPHandler returns an http.Handler that serves the six Store operations
// over JSON, delegating to the wrapped store:
//
//	POST /bind      body {kind,name,id}     -> 204 on success
//	GET  /resolve?kind=..&name=..           -> 200 {found,id}
//
// The request context is passed through to the underlying Store.
func NewHTTPHandler(store Store) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/bind", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req bindReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request body", http.StatusBadRequest)
			return
		}
		id, err := decodeID(req.ID)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		switch req.Kind {
		case "ref":
			err = store.BindRef(r.Context(), RefName(req.Name), id)
		case "alias":
			err = store.BindAlias(r.Context(), Alias(req.Name), id)
		case "projection":
			err = store.BindProjection(r.Context(), ProjectionHandle(req.Name), id)
		default:
			http.Error(w, "unknown kind", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req bindReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request body", http.StatusBadRequest)
			return
		}
		if req.Kind != "ref" {
			http.Error(w, "only ref deletion is supported", http.StatusBadRequest)
			return
		}
		if err := store.DeleteRef(r.Context(), RefName(req.Name)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/resolve", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		kind := r.URL.Query().Get("kind")
		name := r.URL.Query().Get("name")
		var (
			id    identity.VertexID
			found bool
		)
		switch kind {
		case "ref":
			id, found = store.ResolveRef(r.Context(), RefName(name))
		case "alias":
			id, found = store.ResolveAlias(r.Context(), Alias(name))
		case "projection":
			id, found = store.ResolveProjection(r.Context(), ProjectionHandle(name))
		default:
			http.Error(w, "unknown kind", http.StatusBadRequest)
			return
		}
		resp := resolveResp{Found: found}
		if found {
			resp.ID = encodeID(id)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	return mux
}

// HTTPStore is a Store backed by a remote NewHTTPHandler server. Bindings are
// sent as POST /bind; resolutions as GET /resolve. The caller's context is
// attached to every request.
//
// Resolution has no error return (per the Store interface), so a transport
// failure surfaces as "not found" (ok == false) rather than an error.
type HTTPStore struct {
	base   string
	client *http.Client
}

var _ Store = (*HTTPStore)(nil)

// NewHTTPStore returns a Store that talks to the server at base. A nil client
// uses http.DefaultClient.
func NewHTTPStore(base string, client *http.Client) *HTTPStore {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPStore{base: strings.TrimRight(base, "/"), client: client}
}

func (s *HTTPStore) bind(ctx context.Context, kind, name string, id identity.VertexID) error {
	body, err := json.Marshal(bindReq{Kind: kind, Name: name, ID: encodeID(id)})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/bind", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("namespace: remote bind %s/%q failed: %s", kind, name, resp.Status)
	}
	return nil
}

func (s *HTTPStore) resolve(ctx context.Context, kind, name string) (identity.VertexID, bool) {
	u := fmt.Sprintf("%s/resolve?kind=%s&name=%s", s.base, url.QueryEscape(kind), url.QueryEscape(name))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return identity.VertexID{}, false
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return identity.VertexID{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return identity.VertexID{}, false
	}
	var rr resolveResp
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil || !rr.Found {
		return identity.VertexID{}, false
	}
	id, err := decodeID(rr.ID)
	if err != nil {
		return identity.VertexID{}, false
	}
	return id, true
}

// BindRef satisfies Store.
func (s *HTTPStore) BindRef(ctx context.Context, name RefName, id identity.VertexID) error {
	return s.bind(ctx, "ref", string(name), id)
}

// ResolveRef satisfies Store.
func (s *HTTPStore) ResolveRef(ctx context.Context, name RefName) (identity.VertexID, bool) {
	return s.resolve(ctx, "ref", string(name))
}

// DeleteRef satisfies Store.
func (s *HTTPStore) DeleteRef(ctx context.Context, name RefName) error {
	body, err := json.Marshal(bindReq{Kind: "ref", Name: string(name)})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/delete", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("namespace: remote delete ref %q failed: %s", name, resp.Status)
	}
	return nil
}

// BindAlias satisfies Store.
func (s *HTTPStore) BindAlias(ctx context.Context, name Alias, id identity.VertexID) error {
	return s.bind(ctx, "alias", string(name), id)
}

// ResolveAlias satisfies Store.
func (s *HTTPStore) ResolveAlias(ctx context.Context, name Alias) (identity.VertexID, bool) {
	return s.resolve(ctx, "alias", string(name))
}

// BindProjection satisfies Store.
func (s *HTTPStore) BindProjection(ctx context.Context, name ProjectionHandle, id identity.VertexID) error {
	return s.bind(ctx, "projection", string(name), id)
}

// ResolveProjection satisfies Store.
func (s *HTTPStore) ResolveProjection(ctx context.Context, name ProjectionHandle) (identity.VertexID, bool) {
	return s.resolve(ctx, "projection", string(name))
}
