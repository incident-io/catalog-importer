local catalog = import 'catalog.jsonnet';

{
  // Mark all entries as having come from this repo.
  sync_id: 'incident-io/catalog',

  pipelines: [
    // Teams.
    {
      outputs: [
        {
          name: 'Team',
          description: 'Teams in Product Development.',
          type_name: 'Custom["Team"]',
          source: {
            name: 'name',
            external_id: 'external_id',
          },
          attributes: [
            {
              id: 'description',
              name: 'Description',
              type: 'Text',
              source: 'description',
            },
            {
              id: 'goal',
              name: 'Goal',
              type: 'Text',
              source: 'goal',
            },
            {
              id: 'tech_lead',
              name: 'Tech lead',
              type: 'SlackUser',
              source: 'tech_lead',
            },
            {
              id: 'engineering_manager',
              name: 'Engineering manager',
              type: 'SlackUser',
              source: 'engineering_manager',
            },
            {
              id: 'product_manager',
              name: 'Product manager',
              type: 'SlackUser',
              source: 'product_manager',
            },
            {
              id: 'slack_user_group',
              name: 'Slack user group',
              type: 'SlackUserGroup',
              source: 'slack_user_group',
            },
            {
              id: 'slack_channel',
              name: 'Slack channel',
              type: 'String',
              source: 'slack_channel',
            },
            {
              id: 'alert_channel',
              name: 'Alert channel',
              type: 'String',
              source: 'alert_channel',
            },
            {
              id: 'linear_team',
              name: 'Linear team',
              type: 'LinearTeam',
              source: 'linear_team',
            },
            {
              id: 'members',
              name: 'Members',
              type: 'SlackUser',
              array: true,
              source: 'members',
            },
          ],
        },
      ],
      sources: [
        {
          inline: {
            entries: catalog.teams,
          },
        },
      ],
    },

    // Features.
    {
      outputs: [
        {
          name: 'Feature',
          description: 'Product features that would be recognisable to customers.',
          type_name: 'Custom["Feature"]',
          source: {
            name: 'name',
            external_id: 'external_id',
          },
          attributes: [
            {
              id: 'description',
              name: 'Description',
              type: 'Text',
              source: 'description',
            },
            {
              id: 'owner',
              name: 'Owner',
              type: 'Custom["Team"]',
              source: 'owner',
            },
          ],
        },
      ],
      sources: [
        {
          inline: {
            entries: catalog.features,
          },
        },
      ],
    },

    // Integrations.
    {
      outputs: [
        {
          name: 'Integration',
          description: 'Product integrations with third-party services, powering features.',
          type_name: 'Custom["Integration"]',
          source: {
            name: 'name',
            external_id: 'external_id',
          },
          attributes: [
            {
              id: 'description',
              name: 'Description',
              type: 'Text',
              source: 'description',
            },
          ],
        },
      ],
      sources: [
        {
          inline: {
            entries: catalog.integrations,
          },
        },
      ],
    },
  ],
}
