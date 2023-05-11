// Loads data from Backstage.
{
  // TODO: Choose how to source your data, uncommenting/removing as necessary.
  sources: [
    // This points at the local catalog-info.json file. It is only for
    // demonstration purposes, and can be replaced by either of the following
    // examples.
    //
    // https://github.com/incident-io/catalog-importer/blob/master/docs/sources.md#local
    {
      'local': {
        files: [
          'catalog-info.json',
        ],
      },
    },

    // This uses a command to build the catalog entries from the local
    // repository. It's just an example and will need tailoring, depending on
    // how you'd like to source the data.
    //
    // https://github.com/incident-io/catalog-importer/blob/master/docs/sources.md#backstage
    /*
    {
      exec: {
        command: [
          './build-catalog',
        ],
      },
    },
    */

    // Alternatively, you can point the importer directly at your Backstage API
    // and have it pull the entries.
    //
    // https://github.com/incident-io/catalog-importer/blob/master/docs/sources.md#backstage
    /*
    {
      backstage: {
        endpoint: 'https://backstage-internal.example.com/api/catalog/entities',
        token: '$(BACKSTAGE_TOKEN)',  // from environment variable
      },
    },
    */
  ],
  outputs: [
    // Backstage API
    // https://backstage.io/docs/features/software-catalog/descriptor-format/#kind-api
    {
      name: 'Backstage API',
      description: 'APIs synced from Backstage.',
      type_name: 'Custom["BackstageAPI"]',
      source: {
        filter: 'apiVersion == "backstage.io/v1alpha1" && kind == "API"',
        name: 'metadata.name',
        external_id: 'metadata.name',
      },
      attributes: [
        {
          id: 'description',
          name: 'Description',
          type: 'Text',
          source: 'metadata.description',
        },
        {
          id: 'api_type',
          name: 'Type',
          type: 'Custom["BackstageAPIType"]',
          source: 'spec.type',
        },
        {
          id: 'owner',
          name: 'Owner',
          type: 'Custom["BackstageGroup"]',
          source: 'spec.owner.replace("group:", "")',
        },
        {
          id: 'lifecycle',
          name: 'Lifecycle',
          type: 'Custom["BackstageLifecycle"]',
          source: 'spec.lifecycle',
        },
        {
          id: 'system',
          name: 'System',
          type: 'Custom["BackstageSystem"]',
          source: 'spec.system',
        },
        {
          id: 'definition',
          name: 'Definition',
          type: 'String',
          source: 'spec.definition',
        },
      ],
    },

    // Backstage Component
    // https://backstage.io/docs/features/software-catalog/descriptor-format/#kind-component
    {
      name: 'Backstage Component',
      description: 'Components synced from Backstage.',
      type_name: 'Custom["BackstageComponent"]',
      source: {
        filter: 'apiVersion == "backstage.io/v1alpha1" && kind == "Component"',
        name: 'metadata.name',
        external_id: 'metadata.name',
      },
      attributes: [
        {
          id: 'description',
          name: 'Description',
          type: 'Text',
          source: 'metadata.description',
        },
        {
          id: 'type',
          name: 'Type',
          type: 'Custom["BackstageComponentType"]',
          source: 'spec.type',
        },
        {
          id: 'lifecycle',
          name: 'Lifecycle',
          type: 'Custom["BackstageLifecycle"]',
          source: 'spec.lifecycle',
        },
        {
          id: 'owner',
          name: 'Owner',
          type: 'Custom["BackstageGroup"]',
          source: 'spec.owner.replace("group:", "")',
        },
        {
          id: 'subcomponent_of',
          name: 'Subcomponent of',
          type: 'Custom["BackstageComponent"]',
          source: 'spec.subcomponentOf',
        },
        {
          id: 'provides_apis',
          name: 'Provides APIs',
          type: 'Custom["BackstageAPI"]',
          array: true,
          source: 'spec.providesApis',
        },
        {
          id: 'consumes_apis',
          name: 'Consumes APIs',
          type: 'Custom["BackstageAPI"]',
          array: true,
          source: 'spec.consumesApis',
        },
        {
          id: 'depends_on',
          name: 'Depends on',
          type: 'Custom["BackstageComponent"]',
          array: true,
          source: 'spec.dependsOn',
        },
        {
          id: 'tags',
          name: 'Tags',
          array: true,
          source: 'metadata.tags',
          enum: {
            name: 'Backstage Tag',
            type_name: 'Custom["BackstageTag"]',
            description: 'Component tags for searching.',
          },
        },
      ],
    },


    // Backstage Domain
    // https://backstage.io/docs/features/software-catalog/descriptor-format/#kind-domain
    {
      name: 'Backstage Domain',
      description: 'Groups of systems that share terminology or purpose.',
      type_name: 'Custom["BackstageDomain"]',
      source: {
        filter: 'apiVersion == "backstage.io/v1alpha1" && kind == "Domain"',
        name: 'metadata.name',
        external_id: 'metadata.name',
      },
      attributes: [
        {
          id: 'description',
          name: 'Description',
          type: 'Text',
          source: 'metadata.description',
        },
        {
          id: 'owner',
          name: 'Owner',
          type: 'Custom["BackstageGroup"]',
          source: 'spec.owner.replace("group:", "")',
        },
      ],
    },

    // Backstage Group
    // https://backstage.io/docs/features/software-catalog/descriptor-format/#kind-group
    {
      name: 'Backstage Group',
      description: 'Groups syned from Backstage.',
      type_name: 'Custom["BackstageGroup"]',
      source: {
        filter: 'apiVersion == "backstage.io/v1alpha1" && kind == "Group"',
        name: 'metadata.name',
        external_id: 'metadata.name',
      },
      attributes: [
        {
          id: 'type',
          name: 'Type',
          source: 'spec.type',
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
          source: 'spec.parent',
        },
      ],
    },

    // Backstage User
    // https://backstage.io/docs/features/software-catalog/descriptor-format/#kind-user
    {
      name: 'Backstage User',
      description: 'Users syned from Backstage.',
      type_name: 'Custom["BackstageUser"]',
      source: {
        filter: 'apiVersion == "backstage.io/v1alpha1" && kind == "User"',
        name: 'spec.profile.displayName',
        external_id: 'metadata.name',
      },
      attributes: [
        {
          id: 'email',
          name: 'Email',
          type: 'String',
          source: 'spec.profile.email',
        },
        {
          id: 'groups',
          name: 'Groups',
          type: 'Custom["BackstageGroup"]',
          source: 'spec.memberOf',
          array: true,
        },
      ],
    },

    // Backstage System
    // https://backstage.io/docs/features/software-catalog/descriptor-format/#kind-system
    {
      name: 'Backstage System',
      description: 'Collections of resources.',
      type_name: 'Custom["BackstageSystem"]',
      source: {
        filter: 'apiVersion == "backstage.io/v1alpha1" && kind == "System"',
        name: 'metadata.name',
        external_id: 'metadata.name',
      },
      attributes: [
        // Default fields
        {
          id: 'description',
          name: 'Description',
          type: 'Text',
          source: 'metadata.description',
        },
        {
          id: 'owner',
          name: 'Owner',
          type: 'Custom["BackstageGroup"]',
          source: 'spec.owner.replace("group:", "")',
        },
        {
          id: 'domain',
          name: 'Domain',
          type: 'Custom["BackstageDomain"]',
          source: 'spec.domain',
        },
      ],
    },
  ],
}
