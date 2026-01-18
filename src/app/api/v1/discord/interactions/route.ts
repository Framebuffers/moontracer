import { handleCommand } from '@/app/discord/handlers/commandHandler';
import { verifyDiscordRequest } from '@/app/discord/utils/verify';
import { getSecretSync } from '@/app/lib/secrets';

export async function POST(request: Request) {
  const testSecret = request.headers.get('x-test-secret');
  const apiSecret = getSecretSync('api_secret_key');
  if (testSecret === apiSecret) {
    const body = await request.json();
    return handleInteraction(body);
  }

  // Production - verify signature with raw body
  const rawBody = await request.text(); 
  const signature = request.headers.get('x-signature-ed25519');
  const timestamp = request.headers.get('x-signature-timestamp');
  
  if (!signature || !timestamp) {
    return new Response('Missing signature headers', { status: 401 });
  }

  const isValid = verifyDiscordRequest(rawBody, signature, timestamp);
  if (!isValid) {
    return new Response('Invalid request signature', { status: 401 });
  }

  const body = JSON.parse(rawBody);
  return handleInteraction(body);
}

async function handleInteraction(body: any) {
  if (!body || typeof body !== 'object') {
    return new Response('Invalid request body', { status: 400 });
  }

  // PING
  if (body.type === 1) {
    return Response.json({ type: 1 });
  }

  // APPLICATION_COMMAND
  if (body.type === 2) {
    try {
      const result = await handleCommand(body);
      return Response.json(result);
    } catch (error) {
      console.error('Command execution failed:', error);
      return Response.json({
        type: 4,
        data: {
          content: '❌ Command failed to execute',
          flags: 64, // Ephemeral
        },
      });
    }
  }

  return new Response('Unknown interaction type', { status: 400 });
}