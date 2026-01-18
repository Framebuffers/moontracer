import { NextResponse } from 'next/server';
import { verifyDiscordRequest } from '@/app/discord/utils/verify';
import { handleCommand } from '@/app/discord/handlers/commandHandler';

export async function POST(request: Request) {
  try {
    const isValid = await verifyDiscordRequest(request);
    if (!isValid) {
      return NextResponse.json({ error: 'Invalid signature' }, { status: 401 });
    }

    const contentType = request.headers.get('content-type');
    if (!contentType?.includes('application/json')) {
      return NextResponse.json({
        error: 'Content must be application/json'
      }, {status: 400});
    }

    const body = await request.json();
    if (!body || body.trim() === '') {
      return NextResponse.json({
        error: 'Empty request body',
        hint: 'There should be a Discord payload here. It is a mystery.'
      }, { status: 400 });
    }

    switch (body.type) {
      case body.type === 1:
        // verification ping
        return NextResponse.json({ type: 1 });
      case body.type === 2:
        // slash command
        const response = await handleCommand(body);
        return NextResponse.json(response);
      // case body.type === 3:
      //   // button
      //   // return handleButton(body);
      //   return;
      // case body.type === 5:
      //   // modal submit
      //   // return handleModal(body);
        // return;
      default:
        return NextResponse.json({ error: 'Unknown interaction type.' }, { status: 400 });
    }
  } catch (error) {
    console.error('Failed to process command: ', error);
    return NextResponse.json({
      error: error instanceof Error ? error.message : 'Unknown error'
    }, { status: 500 });
  }
}

export async function GET() {
  return NextResponse.json({
    message: 'Discord interactions endpoint',
    method: 'POST only',
    path: '/api/v1/discord/interactions',
    interactions: ['slash_commands', 'buttons', 'modals']
  });
};