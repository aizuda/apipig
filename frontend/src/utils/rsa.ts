function decodeBase64(value: string): Uint8Array<ArrayBuffer> {
  const binary = globalThis.atob(value)
  const bytes = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }
  return bytes
}

async function createWebCryptoEncryptor(publicKey: string, cryptoApi: Crypto) {
  const key = await cryptoApi.subtle.importKey(
    'spki',
    decodeBase64(publicKey),
    { name: 'RSA-OAEP', hash: 'SHA-256' },
    false,
    ['encrypt'],
  )

  return async (value: string): Promise<string> => {
    const ciphertext = await cryptoApi.subtle.encrypt(
      { name: 'RSA-OAEP' },
      key,
      new TextEncoder().encode(value),
    )
    const bytes = new Uint8Array(ciphertext)
    let binary = ''
    for (const byte of bytes) binary += String.fromCharCode(byte)
    return globalThis.btoa(binary)
  }
}

async function createForgeEncryptor(publicKey: string) {
  const forge = await import('node-forge')
  const publicKeyDer = forge.util.decode64(publicKey)
  const publicKeyAsn1 = forge.asn1.fromDer(publicKeyDer)
  const key = forge.pki.publicKeyFromAsn1(publicKeyAsn1)

  return async (value: string): Promise<string> => {
    const ciphertext = key.encrypt(forge.util.encodeUtf8(value), 'RSA-OAEP', {
      md: forge.md.sha256.create(),
      mgf1: { md: forge.md.sha256.create() },
    })
    return forge.util.encode64(ciphertext)
  }
}

export async function createRSAEncryptor(
  publicKey: string,
  cryptoApi: Crypto | undefined = globalThis.crypto,
) {
  if (cryptoApi?.subtle) return createWebCryptoEncryptor(publicKey, cryptoApi)
  if (!cryptoApi?.getRandomValues) {
    throw new Error('Secure random number generation is not supported by this browser')
  }
  return createForgeEncryptor(publicKey)
}
