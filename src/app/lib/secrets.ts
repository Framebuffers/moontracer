// src/app/lib/secrets.ts
import { readFileSync, existsSync } from 'fs';

export function getSecretSync(name: string): string {
  // Production: Read from Docker secrets (mounted at runtime)
  const dockerSecretPath = `/run/secrets/${name}`;
  
  if (existsSync(dockerSecretPath)) {
    try {
      const value = readFileSync(dockerSecretPath, 'utf8').trim();
      console.log(`✓ Loaded secret '${name}' from Docker secrets`);
      return value;
    } catch (error) {
      const err = error as NodeJS.ErrnoException;
      console.error(`✗ Failed to read Docker secret '${name}':`, err.message);
      throw error;
    }
  }
  
  // Development fallback
  if (process.env.NODE_ENV !== 'production') {
    const { join } = require('path');
    const localSecretPath = join(process.cwd(), 'secrets', `${name}.txt`);
    
    if (existsSync(localSecretPath)) {
      const value = readFileSync(localSecretPath, 'utf8').trim();
      console.log(`✓ Loaded secret '${name}' from local dev file`);
      return value;
    }
  }
  
  throw new Error(`Secret '${name}' not found in /run/secrets/ or local dev files`);
}