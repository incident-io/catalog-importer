// Backstage Lifecycle
// https://backstage.io/docs/features/software-catalog/descriptor-format/#speclifecycle-required
{
  sources: [
    {
      inline: {
        entries: [
          {
            name: 'Experimental',
            external_id: 'experimental',
            description:
              'An experiment or early, non-production component, signaling that users may not prefer to consume it over other more established components, or that there are low or no reliability guarantees.',
          },
          {
            name: 'Production',
            external_id: 'production',
            description:
              'An established, owned, maintained component.',
          },
          {
            name: 'Deprecated',
            external_id: 'deprecated',
            description:
              'A component that is at the end of its lifecycle, and may disappear at a later point in time.',
          },
        ],
      },
    },
  ],
  outputs: [
    {
      name: 'Backstage Lifecycle',
      description: 'Component lifecycle stage.',
      type_name: 'Custom["BackstageLifecycle"]',
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
