import { NextResponse } from 'next/server';
import { getSecretSync } from '@/app/lib/secrets';

export async function GET() {
  const secrets: Record<string, boolean> = {};
  const errors: string[] = [];

  const secretNames = ['discord_bot_token', 'discord_public_key', 'database_url', 'api_secret_key'];
  
  for (const name of secretNames) {
    try {
      const value = getSecretSync(name);
      secrets[name] = value.length > 0;
    } catch (error) {
      secrets[name] = false;
      errors.push(`${name}: ${error instanceof Error ? error.message : 'failed'}`);
    }
  }

  const allLoaded = Object.values(secrets).every(loaded => loaded);

  if (!allLoaded) {
    return NextResponse.json({
      status: 'degraded',
      environment: process.env.NODE_ENV || 'development',
      secrets,
      errors,
    }, { status: 500 });
  }

  return NextResponse.json({
    status: 'ok',
    environment: process.env.NODE_ENV || 'development',
    secrets,
  });
}