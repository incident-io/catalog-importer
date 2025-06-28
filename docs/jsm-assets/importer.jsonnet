// Configuration - customize these variables as needed
local objectType = 'Component';
local aqlQuery = 'objectType = Component';

{
  sync_id: "jsm-assets-sync",

  pipelines: [
    {
      sources: [
        {
          exec: {
            // This curl command queries the JSM Assets API using AQL (Assets Query Language)
            // to fetch objects matching the specified query. The result is piped through jq
            // to extract just the "values" array containing the object data.
            command: [
              "bash",
              "-c",
              'curl -u "$JSM_EMAIL:$JSM_API_TOKEN" -H "Accept: application/json" -H "Content-Type: application/json" -d "{\\"qlQuery\\": \\"' + aqlQuery + '\\"}" -X POST https://api.atlassian.com/jsm/assets/workspace/$JSM_WORKSPACE_ID/v1/object/aql | jq ".values"',
            ],
          },
        },
      ],
      outputs: [
        {
          // These fields define how the catalog type appears in incident.io:
          // - name: Shows as the catalog type name in the UI
          // - description: Appears in the catalog type description
          // - type_name: Used internally by incident.io (Custom["JSMComponent"] creates a custom type)
          // Customize these to match your organization's naming conventions
          name: "JSM " + objectType,
          description: objectType + " objects imported from JSM Assets",
          type_name: 'Custom["JSM' + objectType + '"]',
          source: {
            name: "$.name",
            external_id: "$.id",
          },
          attributes: [
            {
              name: "Object Key",
              id: "object_key",
              source: "$.objectKey",
              type: "Text",
            },
          ],
        },
      ],
    },
  ],
}