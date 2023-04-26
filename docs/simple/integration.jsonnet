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
        entries:
          [
            {
              external_id: 'jira',
              name: 'Jira',
              description: 'Use Jira Cloud to export actions, and have incident.io monitor the state of them.',
            },
            {
              external_id: 'jira_server',
              name: 'Jira Server',
              description: 'Use Jira Server to export actions, and have incident.io monitor the state of them.',
            },
            {
              external_id: 'okta',
              name: 'Okta',
              description: 'Connect Okta to incident.io with SCIM and/or SAML to manage your users.',
            },
            {
              external_id: 'zendesk',
              name: 'Zendesk',
              description: 'Use Zendesk to link tickets with incidents, and to push incident updates back into Zendesk.',
            },
            {
              external_id: 'confluence',
              name: 'Confluence',
              description: 'Use Confluence to export post-mortems for archival or collaboration.',
            },
            {
              external_id: 'pagerduty',
              name: 'PagerDuty',
              description: 'Use PagerDuty to escalate during incidents, or auto-create incidents.',
            },
            {
              external_id: 'opsgenie',
              name: 'Opsgenie',
              description: 'Use Opsgenie to escalate during incidents.',
            },
            {
              external_id: 'zoom',
              name: 'Zoom',
              description: 'Use Zoom to automatically start call links for your incident channels, and more.',
            },
            {
              external_id: 'google_docs',
              name: 'Google Docs',
              description: 'Use Google Docs to export post-mortems for collaboration.',
            },
            {
              external_id: 'linear',
              name: 'Linear',
              description: 'Use Linear to export actions and have incident.io monitor the state of them.',
            },
            {
              external_id: 'notion',
              name: 'Notion',
              description: 'Use Notion to export post-mortems for collaboration.',
            },
            {
              external_id: 'shortcut',
              name: 'Shortcut',
              description: 'Use Shortcut to export actions and have incident.io monitor the state of them.',
            },
            {
              external_id: 'splunk_on_call',
              name: 'Splunk On-Call',
              description: 'Use Splunk On-Call to escalate incidents and page policies or people.',
            },
            {
              external_id: 'asana',
              name: 'Asana',
              description: 'Use Asana to export actions and have incident.io monitor the state of them.',
            },
            {
              external_id: 'github',
              name: 'GitHub',
              description: 'Use Github to export actions and have incident.io monitor the\nstate of them.',
            },
            {
              external_id: 'statuspage',
              name: 'Statuspage',
              description: 'Use Statuspage to manage your external Statuspage from within an incident channel.',
            },
            {
              external_id: 'vanta',
              name: 'Vanta',
              description: 'Maintain policy compliance by having incident.io data synced with Vanta.',
            },
            {
              external_id: 'sentry',
              name: 'Sentry',
              description: 'Use Sentry to link issues with incidents, to hear about updates and resolve them when the incident has closed.',
            },
            {
              external_id: 'google_meet',
              name: 'Google Meet',
              description:
                'Use Google Meet to automatically start call links for your incident channels.',
            },
            {
              external_id: 'datadog',
              name: 'Datadog',
              description: 'Use Datadog to see what monitors triggered an incident.',
            },
            {
              external_id: 'slack',
              name: 'Slack',
              description:
                'Slack is the account connected to incident.io, and used to power the incident bot.',
            },
          ],
      },
    },
  ],
}
