// @ts-nocheck -- @codedia/parser emits dynamic data contracts and untyped loop parameters.
// Generated from Notes.aui. Edit the source, then npm run generate.
import { Alert, Badge, Button, Checkbox, Divider, Heading, Icon, Image, Input, Link, Page, Row, Section, Select, Spacer, Stack, Text } from "@/components/ui"

export function Notes({ data, onAction }: { data: any; onAction: (name: string) => void }) {
  return (
    <Page>
      {data.app.user ? (
        <>
          <Row region="topbar">
            <Row region="brand">
              <Button intent="menu" variant="icon" title="Toggle navigation"><Icon name="Menu" /></Button>
              <Image src="./notes.svg" alt="Notes" width={32} height={32} />
              <Heading level={1}>Notes</Heading>
            </Row>
            <Input field="search" variant="search" placeholder="Search your notes" value={data.app.search} />
            <Row region="account">
              <Badge tone={data.app.saveTone}>{data.app.saveStatus}</Badge>
              <Button intent="theme" variant="icon" title="Toggle dark theme"><Icon name="SunMoon" /></Button>
              <Button intent="settings" variant="avatar" title="Settings and account">{data.app.initials}</Button>
            </Row>
          </Row>
          <Row region={data.app.shellRegion}>
            <Section region="sidebar">
              <Button intent="compose" variant="compose">
                <Icon name="Plus" />
                <Text>Compose note</Text>
              </Button>
              {data.app.navigation.map((item, i) => (
                <Button key={i} intent="navigate" payload={item.key} variant={item.variant}>
                  <Icon name={item.icon} />
                  <Text>{item.label}</Text>
                  <Badge>{item.count}</Badge>
                </Button>
              ))}
              <Row region="section-label">
                <Text>NOTEBOOKS</Text>
                <Button intent="new-notebook" variant="icon" title="New notebook"><Icon name="Plus" /></Button>
              </Row>
              <Input field="book-search" variant="small-search" placeholder="Find a notebook" value={data.app.bookSearch} />
              {data.app.books.map((item, i) => (
                <Row key={i} region="notebook-row">
                  <Button intent="notebook" payload={item.id} variant={item.variant}>
                    <Icon name="FolderClosed" />
                    <Text>{item.displayName}</Text>
                  </Button>
                  <Button intent="edit-notebook" payload={item.id} variant="icon" title="Edit notebook"><Icon name="Ellipsis" /></Button>
                </Row>
              ))}
              <Row region="section-label"><Text>TAGS</Text></Row>
              {data.app.tags.map((item, i) => (
                <Button key={i} intent="tag" payload={item.name} variant={item.variant}>
                  <Icon name="Hash" tone={item.color} />
                  <Text>{item.name}</Text>
                </Button>
              ))}
              <Spacer />
              <Row region="sidebar-footer">
                <Icon name="Layers3" />
                <Link href="/">One Workspace</Link>
              </Row>
              <Button intent="settings" variant="nav">
                <Icon name="Settings2" />
                <Text>Settings</Text>
              </Button>
            </Section>
            <Section region="note-list">
              <Row region="list-heading">
                <Stack>
                  <Text tone="eyebrow">YOUR WORKSPACE</Text>
                  <Heading level={2}>{data.app.viewTitle}</Heading>
                </Stack>
                <Badge tone="counter">{data.app.noteCount}</Badge>
              </Row>
              <Row region="list-controls">
                <Checkbox field="select-all" checked={data.app.allSelected} label="Select all visible notes" />
                <Select field="sort" value={data.app.sort} options={data.app.sortOptions} label="Sort notes" />
                <Button intent="refresh" variant="icon" title="Refresh notes"><Icon name="RefreshCw" /></Button>
              </Row>
              {data.app.hasSelected && (
                <Row region="bulk-bar">
                  <Text>{data.app.selectedLabel}</Text>
                  <Button intent="bulk-star" variant="icon" title="Star selected notes"><Icon name="Star" /></Button>
                  <Button intent="bulk-archive" variant="icon" title="Archive selected notes"><Icon name="Archive" /></Button>
                  <Button intent="bulk-trash" variant="icon" title="Trash selected notes"><Icon name="Trash2" /></Button>
                </Row>
              )}
              <Section region="note-cards">
                {data.app.notes.map((item, i) => (
                  <Row key={i} region={item.cardRegion}>
                    <Checkbox field="select-note" payload={item.id} checked={item.selected} label={item.selectLabel} />
                    <Button intent="open-note" payload={item.id} variant="note">
                      <Row region="note-card-heading">
                        <Heading level={3}>{item.displayTitle}</Heading>
                        <Text tone="date">{item.date}</Text>
                      </Row>
                      <Text tone="preview">{item.preview}</Text>
                      <Row region="note-card-footer">
                        {item.badges.map((item, i) => (
                          <Badge key={i} tone={item.color}>{item.label}</Badge>
                        ))}
                      </Row>
                    </Button>
                    <Button intent="quick-star" payload={item.id} variant={item.starVariant} title="Toggle star"><Icon name="Star" /></Button>
                  </Row>
                ))}
                {data.app.empty && (
                  <Stack region="empty-list">
                    <Icon name="FileSearch" />
                    <Heading level={3}>A little room to think</Heading>
                    <Text>Create a note, or try another search.</Text>
                    <Button intent="compose" variant="primary">Create a note</Button>
                  </Stack>
                )}
                {data.app.hasMore && (
                  <Button intent="load-more" variant="secondary">Load more notes</Button>
                )}
              </Section>
              <Row region="list-footer">
                <Icon name="ShieldCheck" />
                <Text>Private by default. Yours to organise.</Text>
              </Row>
            </Section>
            <Section region="editor">
              {data.app.note ? (
                <>
                  <Row region="editor-actions">
                    <Row>
                      <Button intent="back" variant="mobile-back" title="Back to notes"><Icon name="ArrowLeft" /></Button>
                      <Icon name="FileText" tone="muted" />
                      <Text>{data.app.notebookLabel}</Text>
                    </Row>
                    <Row>
                      <Button intent="pin" variant={data.app.pinVariant} title="Pin note"><Icon name="Pin" /></Button>
                      <Button intent="star" variant={data.app.starVariant} title="Star note"><Icon name="Star" /></Button>
                      <Button intent="share" variant="icon" title="Share notebook"><Icon name="Users" /></Button>
                      <Button intent="archive" variant="icon" title="Archive or restore note"><Icon name="Archive" /></Button>
                      <Button intent="trash" variant="icon" title="Move to trash or restore"><Icon name="Trash2" /></Button>
                      <Button intent="fullscreen" variant="icon" title="Toggle full screen editor"><Icon name="Maximize2" /></Button>
                      <Button intent="details" variant="icon" title="Note details and history"><Icon name="Ellipsis" /></Button>
                    </Row>
                  </Row>
                  {data.app.error && (
                    <Alert tone="error">{data.app.error}</Alert>
                  )}
                  {data.app.conflict && (
                    <Row region="recovery">
                      <Text>Your draft is safe. Resolve the conflict before saving.</Text>
                      <Button intent="conflict-copy" variant="secondary">Save draft as a copy</Button>
                      <Button intent="reload-note" variant="secondary">Use server version</Button>
                    </Row>
                  )}
                  {data.app.recovery && (
                    <Row region="recovery">
                      <Text>A local draft is available for this note.</Text>
                      <Button intent="recover" variant="secondary">Recover draft</Button>
                      <Button intent="discard-draft" variant="secondary">Discard draft</Button>
                    </Row>
                  )}
                  {data.app.note.deleted && (
                    <Row region="recovery">
                      <Text>This note is in Trash.</Text>
                      <Button intent="trash" variant="secondary">Restore note</Button>
                      <Button intent="purge" variant="danger">Delete forever</Button>
                    </Row>
                  )}
                  <Row region="editor-tabs">
                    <Button intent="mode-rich" variant={data.app.richVariant}>Rich text</Button>
                    <Button intent="mode-markdown" variant={data.app.markdownVariant}>Markdown</Button>
                    <Button intent="mode-split" variant={data.app.splitVariant}>Split view</Button>
                    <Button intent="mode-preview" variant={data.app.previewVariant}>Preview</Button>
                    <Spacer />
                    <Badge tone="muted">{data.app.permissionLabel}</Badge>
                  </Row>
                  <Row region="format-toolbar">
                    <Button intent="fmt-heading" variant="tool" title="Heading"><Icon name="Heading2" /></Button>
                    <Button intent="fmt-bold" variant="tool" title="Bold"><Icon name="Bold" /></Button>
                    <Button intent="fmt-italic" variant="tool" title="Italic"><Icon name="Italic" /></Button>
                    <Button intent="fmt-strike" variant="tool" title="Strikethrough"><Icon name="Strikethrough" /></Button>
                    <Divider />
                    <Button intent="fmt-list" variant="tool" title="Bullet list"><Icon name="List" /></Button>
                    <Button intent="fmt-task" variant="tool" title="Checklist"><Icon name="ListTodo" /></Button>
                    <Button intent="fmt-quote" variant="tool" title="Quote"><Icon name="Quote" /></Button>
                    <Button intent="fmt-code" variant="tool" title="Code block"><Icon name="Code2" /></Button>
                    <Button intent="fmt-table" variant="tool" title="Table"><Icon name="Table2" /></Button>
                    <Button intent="fmt-link" variant="tool" title="Insert link"><Icon name="Link" /></Button>
                    <Button intent="attach" variant="tool" title="Attach file or image"><Icon name="Paperclip" /></Button>
                    <Spacer />
                    <Button intent="fmt-undo" variant="tool" title="Undo"><Icon name="Undo2" /></Button>
                    <Button intent="fmt-redo" variant="tool" title="Redo"><Icon name="Redo2" /></Button>
                  </Row>
                  <Section region="editor-document">
                    <Input field="title" variant="title" placeholder="Untitled note" value={data.app.note.title} />
                    <Row region="document-meta">
                      <Icon name="Clock3" />
                      <Text>{data.app.modifiedLabel}</Text>
                      <Text>·</Text>
                      <Text>{data.app.wordCount}</Text>
                    </Row>
                    <Input field="body" type="editor" value={data.app.note.body} />
                  </Section>
                  <Row region="editor-status">
                    <Row>
                      <Icon name="CheckCheck" />
                      <Text>{data.app.saveStatus}</Text>
                    </Row>
                    <Text>Markdown, powered by Joplin</Text>
                  </Row>
                </>
              ) : (
                <Stack region="welcome">
                  <Image src="./notes.svg" alt="Notes" width={32} height={32} />
                  <Text tone="eyebrow">A CLEAR SPACE FOR YOUR IDEAS</Text>
                  <Heading level={2}>Make room for a good thought.</Heading>
                  <Text>Notes, plans, and the things you don’t want to forget.</Text>
                  <Button intent="compose" variant="primary">Write your first note</Button>
                </Stack>
              )}
              {data.app.dialog && (
                <Section region="dialog-backdrop">
                  <Section region="dialog">
                    <Row region="dialog-heading">
                      <Heading level={2}>{data.app.dialog.title}</Heading>
                      <Button intent="close-dialog" variant="icon" title="Close dialog"><Icon name="X" /></Button>
                    </Row>
                    <Input type="dialog" field="dialog" />
                  </Section>
                </Section>
              )}
            </Section>
          </Row>
        </>
      ) : (
        <Stack region="signin">
          <Image src="./notes.svg" alt="Notes" width={32} height={32} />
          <Text tone="eyebrow">ONE WORKSPACE</Text>
          <Heading level={1}>A home for your thoughts.</Heading>
          <Text>Your notes, connected to your account.</Text>
          <Link href={data.app.loginURL} variant="primary">Sign in with MyAccounts</Link>
          <Text tone="muted">Running locally? Use the one-time sign-in link printed by MyNotes.exe.</Text>
          {data.app.error && (
            <Alert tone="error">{data.app.error}</Alert>
          )}
        </Stack>
      )}
      {data.app.toast && (
        <Alert region="toast" tone="info">{data.app.toast}</Alert>
      )}
    </Page>
  )
}
