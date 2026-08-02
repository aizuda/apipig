export function buildApiPigConnectionInfo(key: string, url: string) {
  return JSON.stringify({
    _type: 'apipig_channel_conn',
    key,
    url,
  })
}
