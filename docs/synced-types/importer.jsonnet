// Adds custom aliases and a runbook URL to the schedules and escalation paths
// that incident.io syncs, using the data in on-call.jsonnet.
//
// The runbook_url attribute must already exist on the EscalationPath type: add
// it in the incident.io dashboard before you run this.
{
  sync_id: 'example-org/example-repo',
  pipelines: [
    {
      sources: [
        {
          'local': {
            files: ['on-call.jsonnet'],
          },
        },
      ],
      synced_type_outputs: [
        {
          type_name: 'Schedule',
          source: {
            filter: '$.schedule != null',
            // Matches the schedule by its ID, name or synced alias.
            match: '$.schedule',
            custom_aliases: ['$.slack_handle', '$.other_names'],
          },
        },
        {
          type_name: 'EscalationPath',
          source: {
            filter: '$.escalation_path != null',
            match: '$.escalation_path',
            custom_aliases: ['$.slack_handle'],
          },
          attributes: [
            { id: 'runbook_url', source: '$.runbook' },
          ],
        },
      ],
    },
  ],
}
