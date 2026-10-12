import { useStore } from '@nanostores/react'
import { useQuery } from '@tanstack/react-query'
import { endpointURLAtom, tokenAtom } from '~/store'
export function nativeBase(endpoint: string) {
  const url = new URL(endpoint || '/graphql', location.origin)
  url.pathname = url.pathname.replace(/\/graphql\/?$/, '').replace(/\/$/, '') + '/api'
  url.search = ''; url.hash = ''
  return url.toString().replace(/\/$/, '')
}
export type RuntimeMetrics = { rssBytes: number | null; heapBytes: number; goroutines: number; cpuPercent: number | null; processUptimeSeconds: number }
export function useRuntimeMetrics() {
  const endpoint = useStore(endpointURLAtom), token = useStore(tokenAtom)
  return useQuery({
    queryKey: ['native-runtime-metrics', endpoint],
    enabled: !!token,
    queryFn: async ({ signal }): Promise<RuntimeMetrics> => {
      const response = await fetch(nativeBase(endpoint) + '/runtime/metrics', { headers: { Authorization: `Bearer ${token}` }, signal })
      if (!response.ok) { const error = new Error(`运行指标不可用（${response.status}）`); if ([401, 403, 404, 501].includes(response.status)) error.name = "NativeUnavailable"; throw error }
      if (!response.headers.get("content-type")?.includes("application/json")) { const error = new Error("当前后端未提供服务进程指标"); error.name = "NativeUnavailable"; throw error }
      return response.json()
    },
    retry: false,
    refetchInterval: (q) => q.state.error?.name === "NativeUnavailable" ? false : q.state.error ? 10000 : 3000,
  })
}
