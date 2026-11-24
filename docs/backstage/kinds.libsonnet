// Which Backstage kinds to import into incident.io.
//
// Component, System, Domain, Group and User are what you need to route alerts
// and incidents to the team that owns a service, so they're on by default.
//
// To import another kind, uncomment it. Every kind you list is fetched from
// Backstage and synced on each run, and catalogs often hold tens of thousands
// of API entities, so only add API if you'll use it.
local kinds = [
  'Component',
  'System',
  'Domain',
  'Group',
  'User',
  // 'API',
  // 'Resource',
];

{
  kinds: kinds,

  // The Backstage API filter that fetches only these kinds. A key repeated in
  // one filter matches any of its values, so this reads as "kind is any of".
  filter: std.join(',', ['kind=' + std.asciiLower(kind) for kind in kinds]),

  // Whether a kind is imported, for including outputs and attributes that
  // depend on it.
  has(kind):: std.member(kinds, kind),
}
