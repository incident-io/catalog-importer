// Backstage Component Type
// https://backstage.io/docs/features/software-catalog/descriptor-format/#spectype-required
{
  sources: [
    {
      inline: {
        entries: [
          {
            name: 'Service',
            external_id: 'service',
            description:
              'A backend service, typically exposing an API.',
          },
          {
            name: 'Website',
            external_id: 'website',
            description:
              'A website.',
          },
          {
            name: 'Library',
            external_id: 'library',
            description:
              'A software library, such as an npm module or a Java library.',
          },
        ],
      },
    },
  ],
  outputs: [
    {
      name: 'Backstage Component Type',
      description: 'Type of a component.',
      type_name: 'Custom["BackstageComponentType"]',
      categories: ['service'],
      source: {
        name: '$.name',
        external_id: '$.external_id',
      },
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
