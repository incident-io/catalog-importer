// Maps Backstage entities to incident.io catalog types, one output per kind.
//
// sources and kinds come from importer.jsonnet and kinds.libsonnet.
function(sources, kinds)

  // Include items only when a kind is imported, so nothing points at a catalog
  // type that doesn't exist.
  local when(kind, items) = if kinds.has(kind) then items else [];

  // Every entry's external ID is its Backstage namespace and name, such as
  // "default/payments".
  local externalID = '($.metadata.namespace || "default") + "/" + $.metadata.name';

  // Backstage references ignore case, and the ones in relations are lowercased,
  // so each entry also gets its external ID in lowercase as an alias.
  local aliases = ['$.metadata.name', '(%s).toLowerCase()' % externalID];

  // Backstage writes references to other entities as "name", "namespace/name"
  // or "kind:namespace/name". refs() turns each into a lowercase
  // "namespace/name", which matches that alias, and drops references to other
  // kinds.
  local refs(expr, kind) = |||
    _.uniq(_.compact(_.map([].concat(%s || []), function(ref) {
      var m = String(ref).toLowerCase().match(/^(?:([^:]+):)?(?:([^\/]+)\/)?(.+)$/);
      if (!m || (m[1] && m[1] != "%s")) return null;
      return (m[2] || "default") + "/" + m[3];
    })))
  ||| % [expr, std.asciiLower(kind)];
  local ref(expr, kind) = '_.first(%s) || null' % refs(expr, kind);

  {
    sources: sources,
    outputs:
      // Component
      // https://backstage.io/docs/features/software-catalog/descriptor-format/#kind-component
      when('Component', [{
        name: 'Backstage Component',
        description: 'Components synced from Backstage.',
        type_name: 'Custom["BackstageComponent"]',
        categories: ['service'],
        source: {
          filter: '$.kind == "Component"',
          name: '$.metadata.title || $.metadata.name',
          external_id: externalID,
          aliases: aliases,
        },
        attributes: [
          {
            id: 'description',
            name: 'Description',
            type: 'Text',
            source: '$.metadata.description',
          },
          {
            id: 'type',
            name: 'Type',
            type: 'Custom["BackstageComponentType"]',
            source: '$.spec.type',
          },
          {
            id: 'lifecycle',
            name: 'Lifecycle',
            type: 'Custom["BackstageLifecycle"]',
            source: '$.spec.lifecycle',
          },
          {
            id: 'subcomponent_of',
            name: 'Subcomponent of',
            type: 'Custom["BackstageComponent"]',
            source: ref('$.spec.subcomponentOf', 'component'),
          },
          {
            id: 'depends_on',
            name: 'Depends on',
            type: 'Custom["BackstageComponent"]',
            array: true,
            source: refs('$.spec.dependsOn', 'component'),
          },
          {
            id: 'tags',
            name: 'Tags',
            array: true,
            source: '$.metadata.tags',
            enum: {
              name: 'Backstage Tag',
              type_name: 'Custom["BackstageTag"]',
              description: 'Component tags for searching.',
            },
          },
        ] + when('Group', [
          {
            id: 'owner',
            name: 'Owner',
            type: 'Custom["BackstageGroup"]',
            source: ref('$.spec.owner', 'group'),
          },
          // An annotation can point at a Group too. This one names the team that
          // gets paged for the component, when it isn't the owner.
          /*
          {
            id: 'on_call_team',
            name: 'On-call team',
            type: 'Custom["BackstageGroup"]',
            source: ref('$.metadata.annotations["example.com/on-call-team"]', 'group'),
          },
          */
        ]) + when('System', [
          {
            id: 'system',
            name: 'System',
            type: 'Custom["BackstageSystem"]',
            source: ref('$.spec.system', 'system'),
          },
        ]) + when('API', [
          {
            id: 'provides_apis',
            name: 'Provides APIs',
            type: 'Custom["BackstageAPI"]',
            array: true,
            source: refs('$.spec.providesApis', 'api'),
          },
          {
            id: 'consumes_apis',
            name: 'Consumes APIs',
            type: 'Custom["BackstageAPI"]',
            array: true,
            source: refs('$.spec.consumesApis', 'api'),
          },
        ]) + when('Resource', [
          {
            id: 'depends_on_resources',
            name: 'Depends on resources',
            type: 'Custom["BackstageResource"]',
            array: true,
            source: refs('$.spec.dependsOn', 'resource'),
          },
        ]),
      }])

      // System
      // https://backstage.io/docs/features/software-catalog/descriptor-format/#kind-system
      + when('System', [{
        name: 'Backstage System',
        description: 'Collections of components, APIs and resources.',
        type_name: 'Custom["BackstageSystem"]',
        categories: ['product-feature'],
        source: {
          filter: '$.kind == "System"',
          name: '$.metadata.title || $.metadata.name',
          external_id: externalID,
          aliases: aliases,
        },
        attributes: [
          {
            id: 'description',
            name: 'Description',
            type: 'Text',
            source: '$.metadata.description',
          },
        ] + when('Group', [
          {
            id: 'owner',
            name: 'Owner',
            type: 'Custom["BackstageGroup"]',
            source: ref('$.spec.owner', 'group'),
          },
        ]) + when('Domain', [
          {
            id: 'domain',
            name: 'Domain',
            type: 'Custom["BackstageDomain"]',
            source: ref('$.spec.domain', 'domain'),
          },
        ]),
      }])

      // Domain
      // https://backstage.io/docs/features/software-catalog/descriptor-format/#kind-domain
      + when('Domain', [{
        name: 'Backstage Domain',
        description: 'Groups of systems that share terminology or purpose.',
        type_name: 'Custom["BackstageDomain"]',
        categories: ['product-feature'],
        source: {
          filter: '$.kind == "Domain"',
          name: '$.metadata.title || $.metadata.name',
          external_id: externalID,
          aliases: aliases,
        },
        attributes: [
          {
            id: 'description',
            name: 'Description',
            type: 'Text',
            source: '$.metadata.description',
          },
        ] + when('Group', [
          {
            id: 'owner',
            name: 'Owner',
            type: 'Custom["BackstageGroup"]',
            source: ref('$.spec.owner', 'group'),
          },
        ]),
      }])

      // Group
      // https://backstage.io/docs/features/software-catalog/descriptor-format/#kind-group
      + when('Group', [{
        name: 'Backstage Group',
        description: 'Groups synced from Backstage.',
        type_name: 'Custom["BackstageGroup"]',
        categories: ['team'],
        source: {
          filter: '$.kind == "Group"',
          name: '$.spec.profile.displayName || $.metadata.title || $.metadata.name',
          external_id: externalID,
          aliases: aliases,
        },
        attributes: [
          {
            id: 'description',
            name: 'Description',
            type: 'Text',
            source: '$.metadata.description',
          },
          {
            id: 'type',
            name: 'Type',
            source: '$.spec.type',
            enum: {
              name: 'Backstage Group Type',
              type_name: 'Custom["BackstageGroupType"]',
              description: 'Type of Backstage groups.',
            },
          },
          {
            id: 'parent',
            name: 'Parent',
            type: 'Custom["BackstageGroup"]',
            source: ref('$.spec.parent', 'group'),
          },
          {
            id: 'email',
            name: 'Email',
            type: 'String',
            source: '$.spec.profile.email',
          },
          // Set from the dashboard, not from Backstage. When you choose this as
          // the escalation paths attribute for teams, linking an escalation path
          // to a team fills it in, and alerts routed to the group page it.
          {
            id: 'escalation_paths',
            name: 'Escalation paths',
            type: 'EscalationPath',
            array: true,
            schema_only: true,
          },
        ] + when('User', [
          // The Backstage users in this group, from each user's Groups attribute.
          {
            id: 'backstage_users',
            name: 'Backstage users',
            type: 'Custom["BackstageUser"]',
            array: true,
            backlink_attribute: 'groups',
          },
          // Those users' incident.io accounts. Choose this as the members
          // attribute when you make Backstage Group your team type.
          {
            id: 'members',
            name: 'Members',
            type: 'User',
            array: true,
            path: ['backstage_users', 'user'],
          },
        ]),
      }])

      // User
      // https://backstage.io/docs/features/software-catalog/descriptor-format/#kind-user
      + when('User', [{
        name: 'Backstage User',
        description: 'Users synced from Backstage.',
        type_name: 'Custom["BackstageUser"]',
        categories: ['user'],
        source: {
          filter: '$.kind == "User"',
          name: '$.spec.profile.displayName || $.metadata.name',
          external_id: externalID,
          aliases: aliases,
        },
        attributes: [
          // Links each Backstage user to the incident.io user with the same
          // email address.
          {
            id: 'user',
            name: 'incident.io user',
            type: 'User',
            source: '$.spec.profile.email',
          },
        ] + when('Group', [
          // Backstage records membership on the User (spec.memberOf) or on the
          // Group (spec.members). Entities from the Backstage API carry both as
          // memberOf relations.
          {
            id: 'groups',
            name: 'Groups',
            type: 'Custom["BackstageGroup"]',
            array: true,
            source: refs('[].concat($.spec.memberOf || [], _.pluck(_.filter($.relations || [], function(r) { return r.type == "memberOf"; }), "targetRef"))', 'group'),
          },
        ]),
      }])

      // API: off by default, see kinds.libsonnet.
      // https://backstage.io/docs/features/software-catalog/descriptor-format/#kind-api
      + when('API', [{
        name: 'Backstage API',
        description: 'APIs synced from Backstage.',
        type_name: 'Custom["BackstageAPI"]',
        categories: ['service'],
        source: {
          filter: '$.kind == "API"',
          name: '$.metadata.title || $.metadata.name',
          external_id: externalID,
          aliases: aliases,
        },
        attributes: [
          {
            id: 'description',
            name: 'Description',
            type: 'Text',
            source: '$.metadata.description',
          },
          {
            id: 'api_type',
            name: 'Type',
            type: 'Custom["BackstageAPIType"]',
            source: '$.spec.type',
          },
          {
            id: 'lifecycle',
            name: 'Lifecycle',
            type: 'Custom["BackstageLifecycle"]',
            source: '$.spec.lifecycle',
          },
        ] + when('Group', [
          {
            id: 'owner',
            name: 'Owner',
            type: 'Custom["BackstageGroup"]',
            source: ref('$.spec.owner', 'group'),
          },
        ]) + when('System', [
          {
            id: 'system',
            name: 'System',
            type: 'Custom["BackstageSystem"]',
            source: ref('$.spec.system', 'system'),
          },
        ]),
      }])

      // Resource: off by default, see kinds.libsonnet.
      // https://backstage.io/docs/features/software-catalog/descriptor-format/#kind-resource
      + when('Resource', [{
        name: 'Backstage Resource',
        description: 'Infrastructure synced from Backstage, such as databases and queues.',
        type_name: 'Custom["BackstageResource"]',
        categories: ['service'],
        source: {
          filter: '$.kind == "Resource"',
          name: '$.metadata.title || $.metadata.name',
          external_id: externalID,
          aliases: aliases,
        },
        attributes: [
          {
            id: 'description',
            name: 'Description',
            type: 'Text',
            source: '$.metadata.description',
          },
          {
            id: 'type',
            name: 'Type',
            type: 'String',
            source: '$.spec.type',
          },
        ] + when('Group', [
          {
            id: 'owner',
            name: 'Owner',
            type: 'Custom["BackstageGroup"]',
            source: ref('$.spec.owner', 'group'),
          },
        ]) + when('System', [
          {
            id: 'system',
            name: 'System',
            type: 'Custom["BackstageSystem"]',
            source: ref('$.spec.system', 'system'),
          },
        ]),
      }]),
  }
