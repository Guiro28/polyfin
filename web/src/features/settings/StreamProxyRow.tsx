import { useMutation } from '@tanstack/react-query'
import { checkStreamProxy, detectWarp, type ProxyCheck } from '@/api'
import { errorMessage } from '@/format'
import { useI18n } from '@/i18n'
import { Button, Notice, TextInput } from '@/ui'
import { FieldRow } from './parts'
import type { SectionFormApi } from './SectionForm'

/**
 * The proxy of the addons routed through one: its address, a test of the address typed, saved or
 * not, through Cloudflare's trace page, and a search for WARP's local proxy, which fills the field
 * when it finds one.
 */
export default function StreamProxyRow({ form, update, error, limits }: SectionFormApi) {
  const { t } = useI18n()
  const s = t.settings
  const check = useMutation({ mutationFn: (address: string) => checkStreamProxy(address) })
  const detect = useMutation({
    mutationFn: detectWarp,
    onSuccess: (found) => {
      if (found.address !== '') update({ streamProxy: found.address })
    },
  })
  const busy = check.isPending || detect.isPending
  return (
    <FieldRow
      anchor="stream-proxy"
      label={s.streamProxy}
      help={s.streamProxyHelp}
      error={error('stream-proxy')}
    >
      <div className="flex flex-wrap items-center gap-2">
        <TextInput
          value={form.streamProxy}
          onValue={(streamProxy) => {
            check.reset()
            update({ streamProxy })
          }}
          placeholder="socks5://127.0.0.1:40000"
          maxLength={limits('streamProxy').max}
          autoComplete="off"
          spellCheck={false}
          mono
          className="max-w-sm"
        />
        <Button
          size="sm"
          disabled={busy}
          onClick={() => {
            detect.reset()
            check.mutate(form.streamProxy.trim())
          }}
        >
          {check.isPending ? s.streamProxyChecking : s.streamProxyCheck}
        </Button>
        <Button
          size="sm"
          disabled={busy}
          onClick={() => {
            check.reset()
            detect.mutate()
          }}
        >
          {detect.isPending ? s.streamProxyChecking : s.streamProxyDetect}
        </Button>
      </div>
      {check.isSuccess && <CheckNotice check={check.data} server={check.variables === ''} />}
      {check.isError && (
        <Notice tone="danger" className="mt-3">
          {errorMessage(t, check.error)}
        </Notice>
      )}
      {detect.isSuccess && (
        <Notice
          tone={detect.data.address !== '' || detect.data.server.warp ? 'ok' : 'warn'}
          className="mt-3"
        >
          {detect.data.address !== ''
            ? s.streamProxyFound(detect.data.address)
            : detect.data.server.warp
              ? s.streamProxyServerOnWarp(detect.data.server.ip)
              : s.streamProxyNotFound}
        </Notice>
      )}
      {detect.isError && (
        <Notice tone="danger" className="mt-3">
          {errorMessage(t, detect.error)}
        </Notice>
      )}
    </FieldRow>
  )
}

/** What a test found, of the proxy, or of the server's own address when the field was empty. */
function CheckNotice({ check, server }: { check: ProxyCheck; server: boolean }) {
  const { t } = useI18n()
  const s = t.settings
  if (!check.reachable) {
    return (
      <Notice tone="danger" className="mt-3">
        {server ? s.streamProxyServerUnreachable : s.streamProxyUnreachable}
      </Notice>
    )
  }
  const text = server
    ? check.warp
      ? s.streamProxyServerWarp(check.ip)
      : s.streamProxyServerDirect(check.ip)
    : check.warp
      ? s.streamProxyWarp(check.ip)
      : s.streamProxyNoWarp(check.ip)
  return (
    <Notice tone={check.warp || server ? 'ok' : 'info'} className="mt-3">
      {text}
    </Notice>
  )
}
