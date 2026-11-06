import { useEffect, useState } from 'preact/hooks'
import { DataError } from '../data'

export type Loaded<T> = { key: string; status: 'ready'; data: T } | { key: string; status: 'error'; error: DataError }

/** What `load` gives for `key`, once it has; undefined while it loads (and for a moment after the key changes). A failure is a DataError. */
export function useLoad<T>(key: string, load: () => Promise<T>): Loaded<T> | undefined {
  const [state, setState] = useState<Loaded<T>>()
  useEffect(() => {
    let live = true
    load().then(
      (data) => {
        if (live) setState({ key, status: 'ready', data })
      },
      (e: unknown) => {
        if (live) setState({ key, status: 'error', error: e instanceof DataError ? e : new DataError('network', String(e)) })
      },
    )
    return () => {
      live = false
    }
  }, [key])
  return state?.key === key ? state : undefined
}
