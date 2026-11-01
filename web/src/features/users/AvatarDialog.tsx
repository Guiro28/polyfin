import { PencilSimpleIcon, TrashIcon, UploadSimpleIcon } from '@phosphor-icons/react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useRef, useState, type ChangeEvent } from 'react'
import {
  deleteUserImage,
  fetchXperienceAvatars,
  queryClient,
  queryKeys,
  setUserImage,
  setXperienceImage,
} from '@/api'
import { errorMessage } from '@/format'
import { useI18n } from '@/i18n'
import { Avatar, Button, cx, FieldError, Modal, Select, TextInput, useToast } from '@/ui'

/** The largest picture sent, as the server keeps it (accounts.MaxImageBytes). */
const maxImageBytes = 5 << 20

/**
 * An avatar chosen before its user exists (the creation form): a picture sent, in base64 with
 * its preview, or an avatar of Xperience's gallery.
 */
export type AvatarChoice =
  { kind: 'upload'; data: string; preview: string } | { kind: 'xperience'; url: string }

/** The address a choice shows as, for its preview. */
export function choicePreview(choice: AvatarChoice): string {
  return choice.kind === 'upload' ? choice.preview : choice.url
}

/** Saves an avatar chosen before the user existed, once created. */
export function saveChoice(user: string, choice: AvatarChoice) {
  return choice.kind === 'upload'
    ? setUserImage(user, choice.data)
    : setXperienceImage(user, choice.url)
}

/** Reads a picture of the device as base64, with its preview. */
function readPicture(file: File): Promise<{ data: string; preview: string }> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const url = String(reader.result)
      resolve({ data: url.slice(url.indexOf(',') + 1), preview: url })
    }
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}

/** An avatar's name, as Xperience's slugs write it, for its title. */
function avatarName(name: string) {
  return (
    name
      .replace(/[-_]+/g, ' ')
      .replace(/\bprofile avatar\b/gi, '')
      .replace(/\s+/g, ' ')
      .trim() || name
  )
}

/**
 * The avatar of a user that opens the dialog to change it, as on Stremfin: an administrator's for
 * anyone (`user`), every user's for their own (`user` null).
 */
export function AvatarButton({
  user,
  name,
  image,
}: {
  user: string | null
  name: string
  image?: string
}) {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        title={t.users.avatar.change}
        aria-label={t.users.avatar.change}
        className="group relative inline-grid cursor-pointer rounded-row"
      >
        <Avatar name={name} image={image} size="lg" />
        <span className="absolute inset-0 grid place-items-center rounded-row bg-black/55 text-white opacity-0 transition-opacity duration-160 group-hover:opacity-100 group-focus-visible:opacity-100">
          <PencilSimpleIcon size={16} aria-hidden="true" />
        </span>
      </button>
      <AvatarDialog
        open={open}
        onClose={() => setOpen(false)}
        name={name}
        image={image}
        target={{ user }}
      />
    </>
  )
}

/**
 * The dialog that changes an avatar, as Stremfin's and Nuvio Manager's: a picture of the device,
 * or one of Xperience's gallery, by category and search, or none. With `target`, a choice is
 * saved at once for that user; with `onPick`, it is handed back, for a user not created yet.
 */
export function AvatarDialog({
  open,
  onClose,
  name,
  image,
  target,
  onPick,
}: {
  open: boolean
  onClose: () => void
  name: string
  image?: string
  target?: { user: string | null }
  onPick?: (choice: AvatarChoice | null) => void
}) {
  const { t } = useI18n()
  const text = t.users.avatar
  const toast = useToast()
  const file = useRef<HTMLInputElement>(null)
  const [category, setCategory] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [tooLarge, setTooLarge] = useState(false)
  const [chosen, setChosen] = useState<string | null>(null)
  const catalog = useQuery({
    queryKey: queryKeys.xperienceAvatars,
    queryFn: ({ signal }) => fetchXperienceAvatars(signal),
    enabled: open,
    staleTime: 5 * 60_000,
  })

  const done = (message: string) => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.users })
    void queryClient.invalidateQueries({ queryKey: queryKeys.session })
    toast(message)
    close()
  }
  const save = useMutation({
    mutationFn: (choice: AvatarChoice) =>
      choice.kind === 'upload'
        ? setUserImage(target!.user, choice.data)
        : setXperienceImage(target!.user, choice.url),
    onSuccess: () => done(text.saved),
    onSettled: () => setChosen(null),
  })
  const remove = useMutation({
    mutationFn: () => deleteUserImage(target!.user),
    onSuccess: () => done(text.removed),
  })
  const busy = save.isPending || remove.isPending

  function close() {
    save.reset()
    remove.reset()
    setTooLarge(false)
    setSearch('')
    onClose()
  }

  function choose(choice: AvatarChoice) {
    if (onPick) {
      onPick(choice)
      close()
      return
    }
    save.mutate(choice)
  }

  async function sent(event: ChangeEvent<HTMLInputElement>) {
    const picture = event.target.files?.[0]
    event.target.value = ''
    if (!picture) return
    if (picture.size > maxImageBytes) {
      setTooLarge(true)
      return
    }
    setTooLarge(false)
    choose({ kind: 'upload', ...(await readPicture(picture)) })
  }

  const avatars = catalog.data?.avatars ?? []
  const categories = catalog.data?.categories ?? []
  // The gallery opens on its first category, avatars of none last.
  const shownCategory = category ?? categories.find((name) => name !== '') ?? categories[0] ?? ''
  const all = '\u0000all'
  const query = search.trim().toLocaleLowerCase()
  // A search looks through every category.
  const matching = avatars.filter((avatar) =>
    query === ''
      ? category === all || avatar.category === shownCategory
      : `${avatar.name} ${avatar.category}`.toLocaleLowerCase().includes(query),
  )
  const error = save.error ?? remove.error

  return (
    <Modal open={open} onClose={close} title={text.change} width={760}>
      <div className="space-y-5 p-5">
        <div className="flex flex-wrap items-center gap-4">
          {image ? (
            <img
              src={image}
              alt=""
              className="size-[72px] rounded-full object-cover ring-2 ring-accent/30"
            />
          ) : (
            <div className="grid size-[72px] place-items-center rounded-full bg-s3 text-[26px] font-semibold text-ink-2 ring-2 ring-accent/30">
              {name.slice(0, 1).toUpperCase()}
            </div>
          )}
          <div className="min-w-[200px] flex-1">
            <p className="font-semibold text-ink">{name}</p>
            <p className="text-small text-ink-3">{text.intro}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <input
              ref={file}
              type="file"
              accept="image/jpeg,image/png,image/webp"
              hidden
              onChange={(event) => void sent(event)}
            />
            <Button
              size="sm"
              icon={UploadSimpleIcon}
              disabled={busy}
              onClick={() => file.current?.click()}
            >
              {text.upload}
            </Button>
            {(image || onPick) && (
              <Button
                size="sm"
                variant="danger"
                icon={TrashIcon}
                disabled={busy}
                loading={remove.isPending}
                onClick={() => (onPick ? (onPick(null), close()) : remove.mutate())}
              >
                {onPick ? text.clear : text.remove}
              </Button>
            )}
          </div>
        </div>
        {tooLarge && <FieldError>{text.tooLarge}</FieldError>}
        {error && <FieldError>{errorMessage(t, error)}</FieldError>}

        <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 border-t border-line pt-4">
          <h3 className="text-[15px] font-semibold text-ink">{text.gallery}</h3>
          <p className="text-micro text-ink-3">
            {text.sourceBefore}{' '}
            <a
              href={catalog.data?.sourceUrl ?? 'https://xperience-app.com/avatars'}
              target="_blank"
              rel="noopener noreferrer"
              className="text-link underline-offset-2 hover:underline"
            >
              {text.sourceLink} ↗
            </a>
            . {text.sourceAfter}
          </p>
        </div>
        <div className="grid grid-cols-2 gap-3 max-sm:grid-cols-1">
          <Select
            aria-label={text.category}
            value={category === all ? all : shownCategory}
            disabled={catalog.isPending || categories.length === 0}
            onValue={setCategory}
            options={[
              { value: all, label: text.allCategories },
              ...categories.map((name) => ({
                value: name,
                label: `${name || text.uncategorized} (${avatars.filter((a) => a.category === name).length})`,
              })),
            ]}
          />
          <TextInput
            type="search"
            value={search}
            onValue={setSearch}
            placeholder={text.searchPlaceholder}
            aria-label={text.search}
            disabled={catalog.isPending}
          />
        </div>
        <p className="text-micro text-ink-3" role="status">
          {catalog.isPending
            ? text.loading
            : catalog.isError
              ? text.unavailable
              : `${text.count(matching.length)}${catalog.data?.copy ? ` · ${text.copy}` : ''}`}
        </p>
        <div
          className={cx(
            'grid max-h-[360px] grid-cols-[repeat(auto-fill,minmax(82px,1fr))] gap-3.5 overflow-y-auto p-1.5 scrollbar-thin max-sm:grid-cols-[repeat(auto-fill,minmax(64px,1fr))]',
            busy && 'pointer-events-none opacity-60',
          )}
        >
          {matching.map((avatar) => {
            const title = avatarName(avatar.name)
            return (
              <button
                key={avatar.id}
                type="button"
                title={title}
                aria-label={text.choose(title)}
                onClick={() => {
                  setChosen(avatar.url)
                  choose({ kind: 'xperience', url: avatar.url })
                }}
                className={cx(
                  'aspect-square w-full cursor-pointer justify-self-center overflow-hidden rounded-full border-2 bg-s2 transition-[border-color,box-shadow] duration-160 hover:border-accent [content-visibility:auto]',
                  chosen === avatar.url
                    ? 'border-accent shadow-[0_0_0_3px] shadow-accent/30'
                    : 'border-line',
                )}
              >
                <img
                  src={avatar.url}
                  alt=""
                  loading="lazy"
                  decoding="async"
                  referrerPolicy="no-referrer"
                  className="size-full object-cover"
                />
              </button>
            )
          })}
          {!catalog.isPending && matching.length === 0 && (
            <p className="col-span-full text-small text-ink-3">{text.none}</p>
          )}
        </div>
      </div>
    </Modal>
  )
}
