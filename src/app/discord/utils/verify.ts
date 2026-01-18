import nacl from 'tweetnacl';
import { getSecretSync } from '@/app/lib/secrets';

export function verifyDiscordRequest(
  rawBody: string,
  signature: string,
  timestamp: string
): boolean {
  try {
    const publicKey = getSecretSync('discord_public_key');
    const message = timestamp + rawBody;
    const isValid = nacl.sign.detached.verify(
      Buffer.from(message, 'utf-8'),
      Buffer.from(signature, 'hex'),
      Buffer.from(publicKey, 'hex')
    );
    
    return isValid;
  } catch (error) {
    console.error('Signature verification error:', error);
    return false;
  }
}
