/** The first 4 bytes of the SHA-256 of the text, in lowercase hex: the sha8 of the ids that `lct` gives (glossary/<sha8(kr)>/<sha8(zh)>). */
export async function sha8(text: string): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(text))
  return [...new Uint8Array(digest).slice(0, 4)].map((b) => b.toString(16).padStart(2, '0')).join('')
}
