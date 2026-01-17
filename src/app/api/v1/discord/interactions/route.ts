import { NextResponse } from 'next/server';

export async function POST(request: Request) {
  try {
    const bodyText = await request.text();
    if (!bodyText || bodyText.trim() === '') {
      return NextResponse.json({ 
        error: 'Empty request body',
        hint: 'There should be a Discord payload here'
      }, { status: 400 });
    }
    
    const body = JSON.parse(bodyText);
    if (body.type === 1) {
      return NextResponse.json({ type: 1 });
    }
    
    return NextResponse.json({
      type: 4,
      data: { content: 'awoo!' }
    });
    
  } catch (error) {
    console.error('Interaction error:', error);
    return NextResponse.json({ 
      error: error instanceof Error ? error.message : 'Unknown error' 
    }, { status: 500 });
  }
}

export async function GET() {
  return NextResponse.json({
    message: 'Discord interactions endpoint',
    method: 'POST only',
    path: '/api/v1/discord/interactions'
  });
}