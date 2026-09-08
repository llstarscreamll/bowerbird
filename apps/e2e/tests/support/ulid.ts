import { randomBytes } from 'node:crypto';

const ENCODING = '0123456789ABCDEFGHJKMNPQRSTVWXYZ';

export function newUlid(now = Date.now()): string {
  let time = now;
  const chars: string[] = new Array(26);
  for (let i = 9; i >= 0; i -= 1) {
    chars[i] = ENCODING[time % 32];
    time = Math.floor(time / 32);
  }

  const entropy = randomBytes(10);
  let acc = 0;
  let bits = 0;
  let idx = 10;
  for (const byte of entropy) {
    acc = (acc << 8) | byte;
    bits += 8;
    while (bits >= 5 && idx < 26) {
      chars[idx] = ENCODING[(acc >>> (bits - 5)) & 31];
      bits -= 5;
      idx += 1;
    }
  }

  return chars.join('');
}
