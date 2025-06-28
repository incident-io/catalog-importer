// Configuration - customize the object type as needed
local objectType = 'Component';

{
  sync_id: "jsm-assets-sync",

  pipelines: [
    {
      sources: [
        {
          exec: {
            command: [
              "bash",
              "-c",
              'curl -u "$JSM_EMAIL:$JSM_API_TOKEN" -H "Accept: application/json" -H "Content-Type: application/json" -d "{\\"qlQuery\\": \\"objectType = ' + objectType + '\\"}" -X POST https://api.atlassian.com/jsm/assets/workspace/$JSM_WORKSPACE_ID/v1/object/aql | jq ".values"',
            ],
          },
        },
      ],
      outputs: [
        {
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