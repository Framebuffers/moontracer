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

  const body = JSON.parse(rawBody); // Parse after verification
  return handleInteraction(body);
}

function handleInteraction(body: any) {
  if (!body || typeof body !== 'object') {
    return new Response('Invalid request body', { status: 400 });
  }

  // PING
  if (body.type === 1) {
    return Response.json({ type: 1 });
  }

  // APPLICATION_COMMAND
  if (body.type === 2) {
    const token = body.token;
    const applicationId = body.application_id;

    // Defer response
    handleCommandDeferred(body, token, applicationId).catch(console.error);
    return Response.json({ type: 5 });
  }

  return new Response('Unknown interaction type', { status: 400 });
}

async function handleCommandDeferred(
  interaction: any,
  token: string,
  applicationId: string
) {
  try {
    const result = await handleCommand(interaction);
    const webhookUrl = `https://discord.com/api/v10/webhooks/${applicationId}/${token}`;
    
    await fetch(webhookUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        content: result.data.content,
      }),
    });
  } catch (error) {
    console.error('Command execution failed:', error);
    const webhookUrl = `https://discord.com/api/v10/webhooks/${applicationId}/${token}`;
    await fetch(webhookUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        content: '❌ Command failed to execute',
        flags: 64,
      }),
    }).catch(() => {});
  }
}