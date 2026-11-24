package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
)

// seed builds a small fictional engineering organisation, plus apiCount generated API
// entities. Group and User entities deliberately mix backstage.io/v1alpha1 and
// backstage.io/v1beta1, and owner references use every form Backstage allows.
func seed(apiCount int) []entity {
	entities := []entity{}
	add := func(apiVersion, kind string, metadata, spec map[string]any) {
		metadata["namespace"] = "default"
		entities = append(entities, normalise(map[string]any{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata":   metadata,
			"spec":       spec,
		}))
	}

	// Groups: a department with four teams, and SRE nested under Platform.
	groups := []struct {
		name, apiVersion, kind, displayName, parent string
		children                                    []string
		members                                     []string
	}{
		{"engineering", "backstage.io/v1alpha1", "department", "Engineering", "", []string{"platform", "payments", "growth", "data"}, nil},
		{"platform", "backstage.io/v1beta1", "team", "Platform", "engineering", []string{"sre"}, nil},
		{"sre", "backstage.io/v1beta1", "team", "Site Reliability", "platform", []string{}, nil},
		{"payments", "backstage.io/v1alpha1", "team", "Payments", "engineering", []string{}, nil},
		// Growth lists its members on the Group rather than on each User.
		{"growth", "backstage.io/v1beta1", "team", "Growth", "engineering", []string{}, []string{"user:default/sue", "mo"}},
		{"data", "backstage.io/v1alpha1", "team", "Data", "engineering", []string{}, nil},
	}
	for _, g := range groups {
		spec := map[string]any{
			"type":     g.kind,
			"children": g.children,
			"profile": map[string]any{
				"displayName": g.displayName,
				"email":       g.name + "-team@example.com",
			},
		}
		if g.parent != "" {
			spec["parent"] = g.parent
		}
		if g.members != nil {
			spec["members"] = g.members
		}
		add(g.apiVersion, "Group", map[string]any{
			"name":        g.name,
			"title":       g.displayName,
			"description": "The " + g.displayName + " team.",
		}, spec)
	}

	users := []struct {
		name, apiVersion, displayName string
		memberOf                      []string
	}{
		{"ed", "backstage.io/v1alpha1", "Ed Amame", []string{"payments"}},
		{"flo", "backstage.io/v1beta1", "Flo Rentine", []string{"group:default/payments"}},
		{"bea", "backstage.io/v1beta1", "Bea Troot", []string{"platform", "sre"}},
		{"al", "backstage.io/v1alpha1", "Al Dente", []string{"platform"}},
		{"rob", "backstage.io/v1beta1", "Rob Otto", []string{"sre"}},
		{"sue", "backstage.io/v1beta1", "Sue Flay", []string{}},
		{"mo", "backstage.io/v1alpha1", "Mo Bile", []string{}},
		{"penny", "backstage.io/v1alpha1", "Penny Wise", []string{"data"}},
	}
	for _, u := range users {
		add(u.apiVersion, "User", map[string]any{
			"name": u.name,
			"annotations": map[string]any{
				"github.com/user-login": u.name + "-gh",
			},
		}, map[string]any{
			"memberOf": u.memberOf,
			"profile": map[string]any{
				"displayName": u.displayName,
				"email":       u.name + "@example.com",
			},
		})
	}

	for _, d := range [][3]string{
		{"commerce", "group:default/payments", "Everything that takes money from customers."},
		{"infrastructure", "platform", "Shared platform and reliability tooling."},
		{"insights", "data", "Data collection and reporting."},
	} {
		add("backstage.io/v1alpha1", "Domain", map[string]any{"name": d[0], "description": d[2]},
			map[string]any{"owner": d[1]})
	}

	for _, s := range [][4]string{
		{"checkout", "commerce", "payments", "Basket, checkout and order placement."},
		{"billing", "commerce", "group:payments", "Invoices, refunds and the ledger."},
		{"acquisition", "commerce", "growth", "The storefront and sign-up funnel."},
		{"identity", "infrastructure", "platform", "Authentication and the API edge."},
		{"observability", "infrastructure", "sre", "Metrics, logs and alerting."},
		{"analytics", "insights", "data", "Event ingestion and reporting."},
	} {
		add("backstage.io/v1alpha1", "System", map[string]any{"name": s[0], "description": s[3]},
			map[string]any{"owner": s[2], "domain": s[1]})
	}

	// The example.com/on-call-team annotation stands in for an "on-call team" fact that
	// differs from the owner: Platform owns auth-service, but SRE carries its pager.
	components := []struct {
		name, kind, lifecycle, owner, system, onCall, description string
		dependsOn, provides, consumes, tags                       []string
	}{
		{"checkout-api", "service", "production", "group:default/payments", "checkout", "payments", "Takes orders and payment details.",
			[]string{"component:default/payments-gateway", "resource:default/orders-db"}, []string{"checkout-api"}, []string{"payments-gateway-api"}, []string{"go"}},
		{"payments-gateway", "service", "production", "payments", "billing", "payments", "Talks to card processors.",
			[]string{"resource:default/ledger-db"}, []string{"payments-gateway-api"}, nil, []string{"go", "pci"}},
		{"invoicing-worker", "service", "production", "group:payments", "billing", "payments", "Generates and sends invoices.",
			[]string{"component:default/payments-gateway"}, nil, []string{"payments-gateway-api"}, []string{"go"}},
		{"web-storefront", "website", "production", "growth", "acquisition", "growth", "The public storefront.",
			[]string{"component:default/checkout-api"}, nil, []string{"checkout-api"}, []string{"typescript"}},
		{"auth-service", "service", "production", "platform", "identity", "sre", "Issues and checks session tokens.",
			[]string{"resource:default/user-store"}, []string{"auth-api"}, nil, []string{"go"}},
		{"api-gateway", "service", "production", "group:default/platform", "identity", "sre", "Routes public traffic to services.",
			[]string{"component:default/auth-service"}, nil, []string{"auth-api"}, []string{"envoy"}},
		{"metrics-pipeline", "service", "experimental", "sre", "observability", "sre", "Collects and forwards metrics.",
			nil, nil, nil, []string{"go"}},
		{"events-ingest", "service", "production", "data", "analytics", "data", "Receives product analytics events.",
			[]string{"resource:default/events-queue"}, nil, nil, []string{"python"}},
		{"reporting-ui", "website", "deprecated", "data", "analytics", "", "Legacy reporting dashboards.",
			[]string{"component:default/events-ingest"}, nil, nil, []string{"typescript"}},
		{"go-common", "library", "production", "platform", "", "", "Shared Go libraries.",
			nil, nil, nil, []string{"go"}},
	}
	for _, c := range components {
		annotations := map[string]any{
			"backstage.io/source-location": "url:https://github.com/example/" + c.name + "/",
			"github.com/project-slug":      "example/" + c.name,
		}
		if c.onCall != "" {
			annotations["example.com/on-call-team"] = c.onCall
		}
		spec := map[string]any{
			"type":      c.kind,
			"lifecycle": c.lifecycle,
			"owner":     c.owner,
		}
		if c.system != "" {
			spec["system"] = c.system
		}
		for key, value := range map[string][]string{"dependsOn": c.dependsOn, "providesApis": c.provides, "consumesApis": c.consumes} {
			if value != nil {
				spec[key] = value
			}
		}
		add("backstage.io/v1alpha1", "Component", map[string]any{
			"name":        c.name,
			"description": c.description,
			"tags":        c.tags,
			"annotations": annotations,
		}, spec)
	}

	for _, r := range [][4]string{
		{"orders-db", "database", "payments", "checkout"},
		{"ledger-db", "database", "group:default/payments", "billing"},
		{"events-queue", "queue", "data", "analytics"},
		{"user-store", "database", "platform", "identity"},
	} {
		add("backstage.io/v1alpha1", "Resource", map[string]any{"name": r[0]},
			map[string]any{"type": r[1], "owner": r[2], "system": r[3]})
	}

	// Named APIs referenced by components, then the generated bulk.
	for _, a := range [][3]string{
		{"checkout-api", "payments", "checkout"},
		{"payments-gateway-api", "payments", "billing"},
		{"auth-api", "platform", "identity"},
	} {
		add("backstage.io/v1alpha1", "API", map[string]any{"name": a[0], "description": "The " + a[0] + "."},
			map[string]any{"type": "openapi", "lifecycle": "production", "owner": a[1], "system": a[2],
				"definition": "openapi: 3.0.0\ninfo:\n  title: " + a[0] + "\n  version: 1.0.0\npaths: {}\n"})
	}
	apiTypes := []string{"openapi", "grpc", "graphql", "asyncapi"}
	apiOwners := []string{"payments", "platform", "growth", "data", "sre"}
	apiSystems := []string{"checkout", "billing", "acquisition", "identity", "observability", "analytics"}
	for i := 1; i <= apiCount; i++ {
		name := fmt.Sprintf("generated-api-%05d", i)
		add("backstage.io/v1alpha1", "API", map[string]any{"name": name, "description": "A generated API."},
			map[string]any{
				"type":       apiTypes[i%len(apiTypes)],
				"lifecycle":  "production",
				"owner":      apiOwners[i%len(apiOwners)],
				"system":     apiSystems[i%len(apiSystems)],
				"definition": "openapi: 3.0.0\ninfo:\n  title: " + name + "\n  version: 1.0.0\npaths: {}\n",
			})
	}

	// Kinds most catalogs have that nobody wants in incident.io.
	add("backstage.io/v1alpha1", "Location", map[string]any{"name": "example-root"},
		map[string]any{"type": "url", "targets": []string{"https://github.com/example/catalog/blob/main/all.yaml"}})
	add("scaffolder.backstage.io/v1beta3", "Template", map[string]any{"name": "go-service", "title": "Go service"},
		map[string]any{"owner": "platform", "type": "service"})

	return entities
}

// loadEntities reads extra entities from a YAML (single or multi-document) or JSON file.
func loadEntities(path string) ([]entity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '[' {
		var list []entity
		if err := json.Unmarshal(trimmed, &list); err != nil {
			return nil, err
		}
		return defaultNamespaces(list), nil
	}

	entities := []entity{}
	for _, doc := range strings.Split("\n"+string(data), "\n---") {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		jsonDoc, err := yaml.YAMLToJSON([]byte(doc))
		if err != nil {
			return nil, err
		}
		var e entity
		if err := json.Unmarshal(jsonDoc, &e); err != nil {
			return nil, err
		}
		if e != nil {
			entities = append(entities, e)
		}
	}

	return defaultNamespaces(entities), nil
}

func defaultNamespaces(entities []entity) []entity {
	for i, e := range entities {
		e = normalise(e)
		metadata, _ := e["metadata"].(map[string]any)
		if metadata == nil {
			metadata = map[string]any{}
			e["metadata"] = metadata
		}
		if _, ok := metadata["namespace"]; !ok {
			metadata["namespace"] = "default"
		}
		entities[i] = e
	}
	return entities
}

// dedupe keeps the last entity for each reference, so an --extra file can replace a
// seeded entity by redefining it.
func dedupe(entities []entity) []entity {
	index := map[string]int{}
	result := []entity{}
	for _, e := range entities {
		if i, ok := index[ref(e)]; ok {
			result[i] = e
			continue
		}
		index[ref(e)] = len(result)
		result = append(result, e)
	}
	return result
}

// withRelations adds the metadata Backstage computes for every entity it serves: a uid,
// an etag and the relations derived from spec fields, in both directions.
func withRelations(entities []entity) []entity {
	relations := map[string][]any{}
	link := func(from, relation, to, inverse string) {
		relations[from] = append(relations[from], map[string]any{"type": relation, "targetRef": to})
		relations[to] = append(relations[to], map[string]any{"type": inverse, "targetRef": from})
	}

	for _, e := range entities {
		self := ref(e)
		spec, _ := e["spec"].(map[string]any)
		if owner, ok := spec["owner"].(string); ok {
			link(self, "ownedBy", parseRef(owner, "group"), "ownerOf")
		}
		if system, ok := spec["system"].(string); ok {
			link(self, "partOf", parseRef(system, "system"), "hasPart")
		}
		if domain, ok := spec["domain"].(string); ok {
			link(self, "partOf", parseRef(domain, "domain"), "hasPart")
		}
		if parent, ok := spec["parent"].(string); ok {
			link(self, "childOf", parseRef(parent, "group"), "parentOf")
		}
		for _, g := range stringList(spec["memberOf"]) {
			link(self, "memberOf", parseRef(g, "group"), "hasMember")
		}
		for _, u := range stringList(spec["members"]) {
			link(parseRef(u, "user"), "memberOf", self, "hasMember")
		}
		for _, d := range stringList(spec["dependsOn"]) {
			link(self, "dependsOn", parseRef(d, "component"), "dependencyOf")
		}
		for _, a := range stringList(spec["providesApis"]) {
			link(self, "providesApi", parseRef(a, "api"), "apiProvidedBy")
		}
		for _, a := range stringList(spec["consumesApis"]) {
			link(self, "consumesApi", parseRef(a, "api"), "apiConsumedBy")
		}
	}

	for _, e := range entities {
		self := ref(e)
		metadata, _ := e["metadata"].(map[string]any)
		metadata["uid"] = uuid.NewSHA1(uuid.NameSpaceURL, []byte(self)).String()
		metadata["etag"] = uuid.NewSHA1(uuid.NameSpaceOID, []byte(self)).String()[:8]
		if rels, ok := relations[self]; ok {
			e["relations"] = rels
		} else {
			e["relations"] = []any{}
		}
	}

	return entities
}

// ref returns the entity's reference in Backstage's kind:namespace/name form.
func ref(e entity) string {
	metadata, _ := e["metadata"].(map[string]any)
	namespace, _ := metadata["namespace"].(string)
	name, _ := metadata["name"].(string)
	return strings.ToLower(kindOf(e)) + ":" + namespace + "/" + name
}

func kindOf(e entity) string {
	kind, _ := e["kind"].(string)
	return kind
}

// parseRef expands a possibly shortened entity reference like "payments" or
// "group:payments" into its full kind:namespace/name form.
func parseRef(s, defaultKind string) string {
	kind := defaultKind
	if k, rest, ok := strings.Cut(s, ":"); ok {
		kind, s = k, rest
	}
	namespace := "default"
	if ns, name, ok := strings.Cut(s, "/"); ok {
		namespace, s = ns, name
	}
	return strings.ToLower(kind) + ":" + namespace + "/" + s
}

func stringList(v any) []string {
	list, _ := v.([]any)
	result := []string{}
	for _, item := range list {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// normalise round-trips through JSON so every entity uses map[string]any and []any,
// whichever Go types built it.
func normalise(v any) entity {
	data, _ := json.Marshal(v)
	var e entity
	_ = json.Unmarshal(data, &e)
	return e
}
