import { getSecretSync } from "@/app/lib/secrets";

export async function verifyDiscordRequest(request: Request): Promise<boolean> {
    const publicKey = getSecretSync('discord_public_key');
    const signature = request.headers.get('x-signature-ed25519');
    const timestamp = request.headers.get('x-signature-timestamp');

    if (!signature || !timestamp) {
        return false;
    }

    if (process.env.NODE_ENV !== "production") {
        return true;
    }

    const { verifyKey } = await import('discord-interactions');
    const body = await request.clone().text();

    return verifyKey(body, signature, timestamp, publicKey);
}