// fake-backstage serves a seeded, in-memory Backstage catalog over the same HTTP
// endpoints the importer's backstage source calls. Use it to try importer configs and
// check documentation examples without a real Backstage instance.
//
//	go run ./cmd/fake-backstage --apis 5000 --auth static --token secret
//
// See README.md in this directory for the flags and the data it seeds.
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/golang-jwt/jwt/v5"
)

var (
	addr           = kingpin.Flag("addr", "Address to listen on").Default("127.0.0.1:7007").String()
	apiCount       = kingpin.Flag("apis", "How many API entities to seed, to mimic a large catalog").Default("500").Int()
	extraFiles     = kingpin.Flag("extra", "YAML or JSON file of extra entities to serve, in the catalog-info.yaml format (repeatable)").Strings()
	authMode       = kingpin.Flag("auth", "Auth to require: none, static (Authorization: Bearer <token>) or legacy (a JWT signed with the base64 --token)").Default("none").Enum("none", "static", "legacy")
	token          = kingpin.Flag("token", "Token for --auth static, or the base64 secret for --auth legacy").Envar("FAKE_BACKSTAGE_TOKEN").String()
	requireHeaders = kingpin.Flag("require-header", "Also require this header, as Name=value, like an auth proxy in front of Backstage would (repeatable)").Strings()
	quiet          = kingpin.Flag("quiet", "Don't log each request").Bool()
)

func main() {
	kingpin.Parse()

	if *authMode != "none" && *token == "" {
		log.Fatalf("--auth %s needs --token", *authMode)
	}

	headers := map[string]string{}
	for _, h := range *requireHeaders {
		name, value, ok := strings.Cut(h, "=")
		if !ok {
			log.Fatalf("--require-header must be Name=value, got %q", h)
		}
		headers[name] = value
	}

	entities := seed(*apiCount)
	for _, path := range *extraFiles {
		extra, err := loadEntities(path)
		if err != nil {
			log.Fatalf("loading %s: %v", path, err)
		}
		entities = append(entities, extra...)
	}
	entities = withRelations(dedupe(entities))
	sort.SliceStable(entities, func(i, j int) bool { return ref(entities[i]) < ref(entities[j]) })

	counts := map[string]int{}
	for _, e := range entities {
		counts[kindOf(e)]++
	}
	log.Printf("serving %d entities: %v", len(entities), counts)

	srv := &server{entities: entities, headers: headers}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/catalog/entities/by-query", srv.authenticated(srv.byQuery))
	mux.HandleFunc("GET /api/catalog/entities", srv.authenticated(srv.list))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprintln(w, "ok") })

	log.Printf("listening on http://%s/api/catalog/entities/by-query (auth=%s)", *addr, *authMode)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

type entity = map[string]any

type server struct {
	entities []entity
	headers  map[string]string
}

func (s *server) authenticated(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.checkAuth(r); err != nil {
			s.logf("%s %s -> 401 (%v)", r.Method, r.URL.RequestURI(), err)
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"error": map[string]any{"name": "AuthenticationError", "message": err.Error()},
			})
			return
		}
		next(w, r)
	}
}

func (s *server) checkAuth(r *http.Request) error {
	for name, value := range s.headers {
		if r.Header.Get(name) != value {
			return fmt.Errorf("missing or wrong %s header", name)
		}
	}

	if *authMode == "none" {
		return nil
	}

	bearer, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return fmt.Errorf("missing Authorization: Bearer header")
	}

	switch *authMode {
	case "static":
		if bearer != *token {
			return fmt.Errorf("static token does not match")
		}
	case "legacy":
		secret, err := base64.StdEncoding.DecodeString(*token)
		if err != nil {
			return fmt.Errorf("server --token is not base64: %w", err)
		}
		parsed, err := jwt.Parse(bearer, func(*jwt.Token) (any, error) { return secret, nil },
			jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
		if err != nil {
			return fmt.Errorf("invalid legacy token: %w", err)
		}
		if sub, _ := parsed.Claims.GetSubject(); sub != "backstage-server" {
			return fmt.Errorf("legacy token sub must be backstage-server, got %q", sub)
		}
	}

	return nil
}

// byQuery implements GET /api/catalog/entities/by-query with cursor pagination.
// https://backstage.io/docs/features/software-catalog/software-catalog-api/#get-entitiesby-query
func (s *server) byQuery(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := intParam(q.Get("limit"), 20)
	offset := 0
	filters := q["filter"]

	// Like Backstage, the cursor carries the original query, so any filter passed
	// alongside it is ignored.
	if c := q.Get("cursor"); c != "" {
		cur, err := decodeCursor(c)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": "invalid cursor"}})
			return
		}
		offset, filters = cur.Offset, cur.Filters
	}

	matched := s.match(filters)
	end := min(offset+limit, len(matched))
	page := matched[min(offset, end):end]

	pageInfo := map[string]any{}
	if end < len(matched) {
		pageInfo["nextCursor"] = encodeCursor(cursor{Offset: end, Filters: filters})
	}

	s.logf("%s %s -> %d items (%d matched)", r.Method, r.URL.RequestURI(), len(page), len(matched))
	writeJSON(w, http.StatusOK, map[string]any{
		"items":      page,
		"totalItems": len(matched),
		"pageInfo":   pageInfo,
	})
}

// list implements the older GET /api/catalog/entities with offset pagination.
// https://backstage.io/docs/features/software-catalog/software-catalog-api/#get-entities
func (s *server) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	matched := s.match(q["filter"])
	offset := min(intParam(q.Get("offset"), 0), len(matched))
	end := len(matched)
	if limit := intParam(q.Get("limit"), 0); limit > 0 {
		end = min(offset+limit, len(matched))
	}

	page := matched[offset:end]
	s.logf("%s %s -> %d items (%d matched)", r.Method, r.URL.RequestURI(), len(page), len(matched))
	writeJSON(w, http.StatusOK, page)
}

func (s *server) match(filters []string) []entity {
	if len(filters) == 0 {
		return s.entities
	}

	parsed := make([]filter, 0, len(filters))
	for _, f := range filters {
		parsed = append(parsed, parseFilter(f))
	}

	matched := []entity{}
	for _, e := range s.entities {
		fields := flatten(e)
		for _, f := range parsed {
			if f.matches(fields) {
				matched = append(matched, e)
				break
			}
		}
	}

	return matched
}

func (s *server) logf(format string, args ...any) {
	if !*quiet {
		log.Printf(format, args...)
	}
}

type cursor struct {
	Offset  int      `json:"offset"`
	Filters []string `json:"filters"`
}

func encodeCursor(c cursor) string {
	data, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeCursor(s string) (cursor, error) {
	var c cursor
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return c, err
	}
	return c, json.Unmarshal(data, &c)
}

func intParam(v string, fallback int) int {
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func init() {
	log.SetFlags(log.Ltime)
	log.SetPrefix("fake-backstage ")
	log.SetOutput(os.Stderr)
}
