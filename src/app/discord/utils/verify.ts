import { getSecretSync } from '@/app/lib/secrets';

export async function verifyDiscordRequest(request: Request): Promise<boolean> {
    // to test the app without using Discord, this enables testing by using curl:
    /*
        curl -X POST https://moontracer.framebuffer.cl/api/v1/discord/interactions \
            -H "Content-Type: application/json" \
            -H "x-test-secret: $API_SECRET" \
            -d '{"type":2,"data":{"name":"ping"}}'
    */
    const testSecret = request.headers.get('x-test-secret');
    const apiSecret = getSecretSync('api_secret_key');

    if (testSecret && testSecret === apiSecret) {
        console.log('Test secret verified - bypassing Discord signature');
        return true;
    }

    const signature = request.headers.get('x-signature-ed25519');
    const timestamp = request.headers.get('x-signature-timestamp');

    if (!signature || !timestamp) {
        return false;
    }

    const publicKey = getSecretSync('discord_public_key');
    const { verifyKey } = await import('discord-interactions');
    const body = await request.clone().text();

    try {
        const isValid = verifyKey(body, signature, timestamp, publicKey);
        if (!isValid) {
            console.error('Invalid Discord signature');
        }
        return isValid;
    } catch (error) {
        console.error('Signature verification error:', error);
        return false;
    }
}