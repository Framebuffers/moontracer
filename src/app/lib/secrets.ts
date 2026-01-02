import { readFileSync, existsSync } from 'fs';
import { join } from 'path';

export function getSecretSync(name: string): string {
  const dockerSecretPath = `/run/secrets/${name}`;
  
  if (existsSync(dockerSecretPath)) {
    try {
      const value = readFileSync(dockerSecretPath, 'utf8').trim();
      console.log(`✓ Loaded secret '${name}' from ${dockerSecretPath}`);
      return value;
    } catch (error) {
      console.error(`✗ Failed to read ${dockerSecretPath}:`, error);
    }
  }
  
  // Fall back to local secrets -- they **must** end in .txt
  const localSecretPath = join(process.cwd(), 'secrets', `${name}.txt`);
  
  if (existsSync(localSecretPath)) {
    try {
      const value = readFileSync(localSecretPath, 'utf8').trim();
      console.log(`✓ Loaded secret '${name}' from ${localSecretPath}`);
      return value;
    } catch (error) {
      console.error(`✗ Failed to read ${localSecretPath}:`, error);
    }
  }
  
  // Neither location worked
  const errorMsg = `Secret '${name}' not found. Checked:\n` +
    `  - ${dockerSecretPath} (${existsSync(dockerSecretPath) ? 'exists but unreadable' : 'not found'})\n` +
    `  - ${localSecretPath} (${existsSync(localSecretPath) ? 'exists but unreadable' : 'not found'})`;
  
  throw new Error(errorMsg);
}