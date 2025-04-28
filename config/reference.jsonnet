{
  // All catalog types created by this config will be marked with this ID.
  //
  // This is how the importer knows which catalog types it should remove when
  // they've been removed from the configuration.
  //
  // A good sync ID would be the repo name that the importer runs from, or the
  // ID of the CI pipeline that runs it.
  sync_id: 'org/repo',

  // Pipelines define a list of sources which load entries, and outputs (catalog
  // types) that we sync the entries into. Pipelines are synced one after the
  // other, and independently.
  pipelines: [
    {
      // List of sources that entries are loaded from. The combined entries from
      // all sources are passed into the outputs.
      sources: [
        // When you have a small number of entries or want to define the entry
        // contents directly in the config file, you can use the inline source.
        {
          inline: {
            // These entries are passed directly into the outputs.
            //
            // Note that outputs still need to map the entry fields to the
            // output attributes, but can do so by referencing the keys (e.g.
            // name, or description).
            entries: [
              {
                external_id: 'some-external-id',
                name: 'Name',
                description: 'Entry description',
              },
            ],
          },
        },
        // Load entries from files on the local filesystem in either YAML, JSON
        // or Jsonnet form.
        {
          'local': {
            // List of file glob patterns to apply from the current directory.
            files: [
              'catalog-info.yaml',
              'pkg/integrations/*/config.yaml',
            ],
          },
        },
        // If you want to pull data directly from Backstage's API.
        {
          backstage: {
            endpoint: 'http://localhost:6969/api/catalog/entities/by-query',
            token: '$(BACKSTAGE_TOKEN)',
          },
        },
        // When catalog data is stored in source code across GitHub repos, use
        // this source to pull the content from files that match patterns inside
        // those repos.
        {
          github: {
            token: '$(GITHUB_TOKEN)',
            repos: [
              'incident-io/*',  // matches all repos under this org
            ],
            files: [
              '**/catalog-info.yaml',
            ],
          },
        },
        // If you need to transform the catalog data, you can use the exec
        // source to run a command.
        //
        // The output of the command must be either Jsonnet, JSON or YAML, and
        // produce either:
        //
        // - map[string]any (an object with string keys)
        // - []map[string]any (a list of object)
        //
        // If a list, we'll return an entry for each element in the list.
        {
          exec: {
            // Use jq to turn an object of key to entry into a list of entries.
            command: [
              'jq',
              'to_entries | map(.value)',
              'catalog.json',
            ],
          },
        },
        // For GraphQL APIs, you can use this source to execute queries and paginate
        // through results.
        //
        // The example shows how you'd query the GitHub GraphQL API to find all the
        // repositories available to the viewer.
        {
          graphql: {
            // The GraphQL endpoint.
            endpoint: 'https://api.github.com/graphql',
            // Headers for authorization or anything else you may need, supporting the
            // credentials from environment variable substitution.
            headers: {
              authorization: 'Bearer $(GITHUB_TOKEN)',
            },
            // This is the query. We support three pagination strategies:
            // - No pagination, where the query has no variables
            // - Use of a $page variable that is iterated once per page, or an $offset
            // that is incremented by the number of results that have been seen
            // - $cursor for cursor based pagination: this requires the
            // paginate.next_cursor to specify where in the GraphQL result you should find
            // the next cursor value
            query: |||
              query($cursor: String) {
              	viewer {
              		repositories(first: 50, after: $cursor) {
              			edges {
              				repository:node {
              					name
              					description
              				}
              			}
              			pageInfo {
              				endCursor
              				hasNextPage
              			}
              		}
              	}
              }
            |||,
            // This is what we pass into the output pipeline.
            result: '$.viewer.repositories.edges',
            // Configure pagination strategies here:
            paginate: {
              // If cursor based, this says where to find the cursor value for the
              // subsequent query.
              next_cursor: '$.viewer.repositories.pageInfo.endCursor',
            },
          },
        },
      ],

      // List of outputs, corresponding to catalog types, that the importer will
      // create and sync entries into.
      //
      // It includes config for the resulting catalog type, along with catalog
      // type attributes and how to build the values of said attributes from the
      // sourced entries.
      //
      // These outputs are passed any entries produced by this pipeline's source
      // list.
      outputs: [
        // This creates a Team catalog type.
        {
          // This is how the type will be displayed in the catalog dashboard.
          name: 'Team',
          description: 'Teams in the Product Development function.',
          // This will determine the group in which this type will appear under. They can be one of the following:
          // 'customer', 'issue-tracker', 'on-call', 'product-feature', 'service', 'team', 'user'
          categories: ['team'],    

          // The unique type name for this catalog type. If other catalog types
          // create attributes that point at this type, they should set the
          // attribute's 'type' to this value.
          //
          // All outputs must use type names of the form `Custom["CamelCase"]`.
          // This is because externally synced catalog types do not have this
          // prefix (e.g. GitHubRepository) and this avoids collisions.
          type_name: 'Custom["Team"]',

          // Control how we filter and map source entries into this output.
          source: {
            // Optionally filter entries provided by this pipeline's source
            // using this field.
            //
            // Only entries where this filter is true will be synced into this
            // output.
            filter: '$.metadata.kind = "Service"',

            // Required mapping of entry field to the external ID of this entry.
            //
            // If your entries are sourced from an external system or have a
            // stable identifier, you can use that ID as an external ID,
            // ensuring that deletion and recreation of this entry will preserve
            // any references from other catalog types.
            //
            // The uuid of the resource in the external catalog system would be
            // an ideal value.
            external_id: '$.metadata.name',

            // Required field in the source that will act as the name for this
            // entry, where name is the human readable label.
            name: '$.metadata.name',
          },

          // Controls which attributes this type will have. Attributes are types
          // and can be single-value or arrays.
          attributes: [
            // Here's a comprehensively documented attribute showing all
            // possible configurations:
            {
              // Stable identifier for this attribute. If this is changed, we'll
              // delete the old attribute and create a new one under the new
              // name.
              id: 'description',

              // The human readable name of this attribute, used as a label in
              // the catalog dashboard. This can be changed provided the id
              // remains the same.
              name: 'Description',

              // Type of this attribute.
              //
              // You can use any of the following primitive values:
              //
              // - String, plain text strings
              // - Text, rich text that supports formatting
              // - Number, floating-point numeric values
              // - Bool, true or false value
              //
              // Or reference other catalog entries by their type name, where
              // that might be:
              //
              // - 'Custom["User"]' for a user type you've synced into the catalog
              // - 'GitHubRepository' for an externally synced catalog type
              // connected to an incident.io integration
              type: 'Text',

              // If this is true, allow zero-or-more values for this attribute.
              array: false,

              // Which field in the sourced entry to use for the value of this
              // attribute.
              //
              // Will default to the id of this attribute.
              source: '$.metadata.description',

              // If true we will only create the attribute in the schema and won't sync
              // the value of the attribute. This is useful when you want to specify the
              // schema but leave this field available to be controlled from the dashboard
              // manually, separately from the importer.
              schema_only: false,
            },

            // Most of the time you can be much less verbose, as source will
            // default to ID, and array is by default false:
            {
              id: 'goal',
              name: 'Goal',
              type: 'Text',
            },

            // Example referencing an externally synced catalog type,
            // LinearTeam:
            {
              id: 'linear_team',
              name: 'Linear team',
              type: 'LinearTeam',  // automatically available if Linear is connected
              source: '$.metadata.annotations["incident.io/linear-team"]',
            },
          ],
        },
      ],
    },
  ],
}
