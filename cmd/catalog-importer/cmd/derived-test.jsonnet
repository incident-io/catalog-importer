{
  sync_id: 'incident-io/laura',
  pipelines: [
    {
      outputs: [
        {
          name: 'Grandparent',
          description: 'Level 1 - Grandparent',
          type_name: 'Custom["Grandparent"]',
          attributes: [
            {
              id: 'name',
              name: 'Name',
              type: 'Text',
            },
            {
              name: 'Children',
              array: true,
              type: 'Custom["Parent"]',
            },
          ],
        },
        {
          name: 'Parent',
          description: 'Level 2 - Parent',
          type_name: 'Custom["Parent"]',
          attributes: [
            {
              name: 'Parent',
              array: true,
              type: 'Custom["Grandparent"]',
            },
            {
              name: 'Children',
              array: true,
              type: 'Custom["Child"]',
            },
          ],
        },
        {
          name: 'Child',
          description: 'Level 3 - Child',
          type_name: 'Custom["Child"]',
          attributes: [
            {
              name: 'Parent',
              array: true,
              type: 'Custom["Parent"]',
            },
            {
              name: 'Children',
              array: true,
              type: 'Custom["Grandchild"]',
            },
          ],
        },
        {
          name: 'Grandchild',
          description: 'Level 4 - Grandchild',
          type_name: 'Custom["Grandchild"]',
          attributes: [
            {
              name: 'Parent',
              array: true,
              type: 'Custom["Child"]',
            },
          ],
        },
      ],
      sources: [
        {
          inline: {
            entries: [
              {
                name: 'Grandpa',
                external_id: 'grandpa',
                description: 'Grandpa',
                type: 'Custom["Grandparent"]',
              },
            ],
          },
        },
      ],
    },
  ],
}
