import { NextResponse } from 'next/server';
import { getSecretSync } from '@/app/lib/secrets';

export async function GET() {
  try {
    const botToken = getSecretSync('discord_bot_token');
    const publicKey = getSecretSync('discord_public_key');
    
    const isValid = botToken.length > 0 && publicKey.length > 0;
    
    return NextResponse.json({
      status: 'ok',
      environment: process.env.NODE_ENV || 'development',
      secretsLoaded: isValid,
      botTokenPrefix: botToken.substring(0, 10) + '...',
      publicKeyPrefix: publicKey.substring(0, 10) + '...',
    });
  } catch (error) {
    return NextResponse.json({
      status: 'error',
      message: error instanceof Error ? error.message : 'Unknown error',
    }, { status: 500 });
  }
}