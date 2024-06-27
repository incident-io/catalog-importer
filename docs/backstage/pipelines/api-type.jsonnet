// Backstage API Type
// https://backstage.io/docs/features/software-catalog/descriptor-format/#spectype-required-2
{
  sources: [
    {
      inline: {
        entries: [
          {
            name: 'OpenAPI',
            external_id: 'openapi',
            description:
              'An API definition in YAML or JSON format based on the OpenAPI version 2 or version 3 spec.',
          },
          {
            name: 'AsyncAPI',
            external_id: 'asyncapi',
            description:
              'An API definition based on the AsyncAPI spec.',
          },
          {
            name: 'GraphQL',
            external_id: 'graphql',
            description:
              'An API definition based on GraphQL schemas for consuming GraphQL based APIs.',
          },
          {
            name: 'gRPC',
            external_id: 'grpc',
            description:
              'An API definition based on Protocol Buffers to use with gRPC.',
          },
        ],
      },
    },
  ],
  outputs: [
    {
      name: 'Backstage API Type',
      description: 'Type or format of the API.',
      type_name: 'Custom["BackstageAPIType"]',
      source: {
        name: '$.name',
        external_id: '$.external_id',
      },
      categories: ['service'],
      attributes: [
        {
          id: 'description',
          name: 'Description',
          type: 'Text',
          source: '$.description',
        },
      ],
    },
  ],
}
