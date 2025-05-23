local catalog = import 'catalog.jsonnet';

{
  // Mark all entries as having come from this repo.
  sync_id: 'incident-io/catalog',

  pipelines: [
    // Teams.
    {
      sources: [
        {
          inline: {
            entries: catalog.teams,
          },
        },
      ],
      outputs: [
        {
          name: 'Team',
          description: 'Teams in Product Development.',
          type_name: 'Custom["Team"]',
          categories: ['team'],
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
            {
              id: 'goal',
              name: 'Goal',
              type: 'Text',
              source: '$.goal',
            },
            {
              id: 'tech_lead',
              name: 'Tech lead',
              type: 'User',
              source: '$.tech_lead',
            },
            {
              id: 'engineering_manager',
              name: 'Engineering manager',
              type: 'User',
              source: '$.engineering_manager',
            },
            {
              id: 'product_manager',
              name: 'Product manager',
              type: 'User',
              source: '$.product_manager',
            },
            {
              id: 'slack_user_group',
              name: 'Slack user group',
              type: 'SlackUserGroup',
              source: '$.slack_user_group',
            },
            {
              id: 'slack_channel',
              name: 'Slack channel',
              type: 'String',
              source: '$.slack_channel',
            },
            {
              id: 'alert_channel',
              name: 'Alert channel',
              type: 'String',
              source: '$.alert_channel',
            },
            {
              id: 'members',
              name: 'Members',
              type: 'User',
              array: true,
              source: '$.members',
            },
            {
              id: 'escalation_path',
              name: 'Escalation path',
              type: 'EscalationPath',
              // This attribute is managed via the incident.io dashboard, to
              // avoid needing to copy data from that dashboard to this config
              // and back.
              schema_only: true,
            },
          ],
        },
      ],
    },

    // Features.
    {
      sources: [
        {
          inline: {
            entries: catalog.features,
          },
        },
      ],
      outputs: [
        {
          name: 'Feature',
          description: 'Product features that would be recognisable to customers.',
          type_name: 'Custom["Feature"]',
          categories: ['product-feature'],
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
            {
              id: 'owner',
              name: 'Owner',
              type: 'Custom["Team"]',
              source: '$.owner',
            },
          ],
        },
      ],
    },

    // Integrations.
    {
      sources: [
        {
          inline: {
            entries: catalog.integrations,
          },
        },
      ],
      outputs: [
        {
          name: 'Integration',
          description: 'Product integrations with third-party services, powering features.',
          type_name: 'Custom["Integration"]',
          categories: ['product-feature'],
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
            {
              id: 'contacts',
              name: 'Contacts',
              array: true,
              source: '$.contacts',
              enum: {
                name: 'Integration Contact',
                type_name: 'Custom["IntegrationContact"]',
                description: 'Contact we have in the company for this integration.',
                enable_backlink: true,
              }
            }
          ],
        },
      ],
    },
  ],
}
