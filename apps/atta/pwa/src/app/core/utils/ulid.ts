/** Crockford Base32 ULID (26 chars). Shared technical util — no domain types. */
const ULID_ALPHABET = '0123456789ABCDEFGHJKMNPQRSTVWXYZ';

export function generateUlid(now: number = Date.now()): string {
  const timePart: string[] = new Array(10);
  let value = now;
  for (let index = 9; index >= 0; index -= 1) {
    timePart[index] = ULID_ALPHABET[value % 32] ?? '0';
    value = Math.floor(value / 32);
  }
  const randomBytes = crypto.getRandomValues(new Uint8Array(10));
  let randomPart = '';
  let bitBuffer = 0;
  let bitCount = 0;
  for (const byte of randomBytes) {
    bitBuffer = (bitBuffer << 8) | byte;
    bitCount += 8;
    while (bitCount >= 5 && randomPart.length < 16) {
      bitCount -= 5;
      randomPart += ULID_ALPHABET[(bitBuffer >> bitCount) & 31] ?? '0';
    }
    if (randomPart.length === 16) break;
  }
  return `${timePart.join('')}${randomPart}`;
}
