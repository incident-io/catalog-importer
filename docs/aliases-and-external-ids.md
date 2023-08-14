# External IDs

In your Catalog, entries have an ID, but they also have an external ID. The ID
is a unique reference in our system, generated automatically for each entry,
which cannot be set or changed. If you ever have something (say the
catalog-importer) sync an entry into a type with an external ID, we’ll interpret
that as the same entry as whatever already exists under that external ID.

In practice, this means:
* You can change the name and any attributes: we attach these to either the
  external ID or the ID of the entry (whichever is available) and any renames
  will transparently apply to custom field values and anywhere else the entry
  was used.
* If you happen to delete an entry but then try recreating it with the same
  external ID, we’ll bring back the archived entry so you preserve any old
  references such as incident custom field values.

# Aliases

The aliases are just about how you might reference a catalog entry via an
attribute on some other entry.

Say you have a team catalog type with a “Slack group” attribute which is of type
“SlackUserGroup”. You could set that value of that attribute to either:
* The ID of the Slack user group catalog entry (e.g. 01H4GFQPTNGNF0FFEGX2BT8811)
* The external ID of the Slack user group catalog entry (e.g. S043VBNSBC3)
* Or any of the aliases (e.g. my-user-group-handle)

Then when you view that team entry in the dashboard or try navigating to its
Slack group attribute, we’ll follow the above rules to find you the match.

Aliases are *only* used for this matching, and nothing else, so:
* They don't show up when you're searching for a catalog entry, either in the
  catalog or to assign it to a custom field
* They can't be edited via the Dashboard UI, only via the API, catalog-importer,
  or Terraform.
