// @ts-nocheck -- @codedia/parser emits dynamic data contracts and untyped loop parameters.
// Generated from Dialog.aui. Edit the source, then npm run generate.
import { Badge, Button, Checkbox, Divider, Field, Heading, Input, Link, Page, Row, Select, Stack, Text } from "@/components/ui"

export function Dialog({ data, onAction }: { data: any; onAction: (name: string) => void }) {
  return (
    <Page>
      {data.app.dialog.isSettings && (
        <Stack region="dialog-content">
          <Text>{data.app.user.name}</Text>
          <Text tone="muted">{data.app.user.email}</Text>
          <Text>Your notes are private unless you share a notebook.</Text>
          <Row>
            <Button intent="theme" variant="secondary">Switch theme</Button>
            <Button intent="logout" variant="secondary">Sign out</Button>
          </Row>
          <Divider />
          <Heading level={3}>Import & export</Heading>
          <Text>Export your notes, tags, notebook hierarchy and attachments. Importing creates new copies.</Text>
          <Row>
            <Button intent="import" variant="secondary">Import notes</Button>
            <Link href={data.app.exportJSON} variant="secondary">JSON backup</Link>
            <Link href={data.app.exportMD} variant="secondary">Markdown ZIP</Link>
            <Link href={data.app.exportHTML} variant="secondary">HTML ZIP</Link>
          </Row>
          <Heading level={3}>Templates</Heading>
          <Text>Mark any note as a template in its details, then duplicate it whenever you need it.</Text>
          <Button intent="show-templates" variant="secondary">Browse templates</Button>
          {data.app.user.admin && (
            <>
              <Divider />
              <Heading level={3}>Administration</Heading>
              <Button intent="admin" variant="secondary">Manage users & system</Button>
            </>
          )}
          <Divider />
          <Text tone="muted">{data.app.versionLabel}</Text>
          <Link href="https://github.com/oneworkspacestudio/notes/releases">Check available releases</Link>
          <Text tone="muted">Joplin Markdown renderer included. Official Joplin sync and E2EE are not available in this preview.</Text>
        </Stack>
      )}
      {data.app.dialog.isNotebook && (
        <Stack region="dialog-content">
          <Field label="Name"><Input field="dialog-name" value={data.app.dialog.name} placeholder="Notebook name" /></Field>
          <Field label="Parent notebook"><Select field="dialog-parent" value={data.app.dialog.parent} options={data.app.parentOptions} /></Field>
          <Text tone="muted">Sharing applies to notes in this notebook; child notebooks have their own permissions.</Text>
          <Row>
            <Button intent="save-notebook" variant="primary">Save notebook</Button>
            {data.app.dialog.editing && (
              <Button intent="delete-notebook" variant="danger">Delete empty notebook</Button>
            )}
          </Row>
        </Stack>
      )}
      {data.app.dialog.isDetails && (
        <Stack region="dialog-content">
          <Field label="Notebook"><Select field="note-notebook" value={data.app.note.notebook_id} options={data.app.detailNotebookOptions} /></Field>
          <Field label="Tags, separated by commas"><Input field="note-tags" value={data.app.tagText} placeholder="work, ideas, personal" /></Field>
          <Field label="Reminder"><Input field="note-reminder" type="datetime-local" value={data.app.reminderValue} /></Field>
          {data.app.hasReminder && (
            <Link href={data.app.calendarURL} variant="secondary">Add reminder to Calendar (.ics)</Link>
          )}
          <Checkbox field="note-template" checked={data.app.note.template} label="Use as a note template" />
          <Heading level={3}>Attachments</Heading>
          {data.app.dialog.attachments.map((item, i) => (
            <Row key={i}>
              <Link href={item.url}>{item.name}</Link>
              <Text tone="muted">{item.sizeLabel}</Text>
              <Button intent="delete-attachment" payload={item.id} variant="danger">Remove</Button>
            </Row>
          ))}
          <Row>
            <Button intent="duplicate" variant="secondary">Duplicate note</Button>
            <Button intent="copy-link" variant="secondary">Copy internal link</Button>
            <Button intent="history" variant="secondary">Version history</Button>
          </Row>
          <Text tone="muted">The last 50 content revisions are retained. Restoring creates another revision.</Text>
          {data.app.dialog.versions.map((item, i) => (
            <Row key={i} region="history-row">
              <Text>{item.label}</Text>
              <Button intent="restore-version" payload={item.revision} variant="secondary">Restore</Button>
            </Row>
          ))}
        </Stack>
      )}
      {data.app.dialog.isShare && (
        <Stack region="dialog-content">
          <Text>Invite someone to this notebook with a single-use link. They must sign in to accept it. Links expire after 24 hours.</Text>
          <Text tone="muted">Only notes directly in this notebook are shared. Private notes outside it remain private.</Text>
          <Select field="dialog-book" value={data.app.dialog.book} options={data.app.ownNotebookOptions} />
          <Select field="dialog-role" value={data.app.dialog.role} options={data.app.roleOptions} />
          <Button intent="create-invite" variant="primary">Create invitation</Button>
          {data.app.dialog.invite && (
            <>
              <Input field="invite" value={data.app.dialog.invite} readOnly={true} />
              <Button intent="copy-invite" variant="secondary">Copy invitation</Button>
            </>
          )}
          {data.app.dialog.members.map((item, i) => (
            <Row key={i}>
              <Text>{item.name}</Text>
              <Badge>{item.role}</Badge>
              <Button intent="revoke-share" payload={item.id} variant="danger">Revoke</Button>
            </Row>
          ))}
        </Stack>
      )}
      {data.app.dialog.isLink && (
        <Stack region="dialog-content">
          <Field label="Link address"><Input field="dialog-href" value={data.app.dialog.href} placeholder="https://example.com or :/note-id" /></Field>
          <Button intent="insert-link" variant="primary">Insert link</Button>
        </Stack>
      )}
      {data.app.dialog.isAdmin && (
        <Stack region="dialog-content">
          <Text>Administrator access manages accounts and limits. It does not grant access to private notes.</Text>
          <Text>{data.app.dialog.system}</Text>
          <Field label="Listen address and port"><Input field="config-listen" value={data.app.dialog.config.listen} /></Field>
          <Field label="Public URL"><Input field="config-public_url" value={data.app.dialog.config.public_url} /></Field>
          <Field label="Storage directory"><Input field="config-data_dir" value={data.app.dialog.config.data_dir} /></Field>
          <Field label="MyAccounts issuer"><Input field="config-oidc_issuer" value={data.app.dialog.config.oidc_issuer} /></Field>
          <Field label="OIDC client ID"><Input field="config-oidc_client_id" value={data.app.dialog.config.oidc_client_id} /></Field>
          <Field label="Session lifetime in hours (1–24)"><Input field="config-session_hours" type="number" value={data.app.dialog.config.session_hours} /></Field>
          <Field label="Maximum upload in MB (1–100)"><Input field="config-upload_mb" type="number" value={data.app.dialog.config.upload_mb} /></Field>
          <Button intent="save-config" variant="secondary">Save configuration for restart</Button>
          <Text tone="muted">Changes require a restart. Moving storage requires an offline backup and restore first. Environment variables supplied by One Workspace take precedence. Set client secrets in the server environment; they are never displayed here.</Text>
          <Text>For a complete system backup or restore, stop the service and use the backup/restore commands. Personal exports are available in Settings.</Text>
          {data.app.dialog.users.map((item, i) => (
            <Row key={i} region="user-row">
              <Stack>
                <Text>{item.name}</Text>
                <Text tone="muted">{item.email}</Text>
              </Stack>
              <Input type="number" field="user-quota" payload={item.id} value={item.quota_mb} label="Quota in MB" />
              <Checkbox field="user-disabled" payload={item.id} checked={item.disabled} label="Disabled" />
              <Button intent="save-user" payload={item.id} variant="secondary">Save</Button>
            </Row>
          ))}
          <Heading level={3}>Audit log</Heading>
          {data.app.dialog.audit.map((item, i) => (
            <Text key={i} tone="muted">{item.label}</Text>
          ))}
          <Link href={data.app.healthURL} variant="secondary">Health status</Link>
        </Stack>
      )}
    </Page>
  )
}
