
import { REST, Routes } from 'discord.js';
import { describe } from 'node:test';

const commands = [
    {
        name: 'ping',
        description: 'pong!'
    },
    {
        name: 'awoo',
        description: 'oh no, don\'t awoo'
    }
]

async function registerCommands() {
    const token = process.env.DISCORD_BOT_TOKEN;
    const applicationId = process.env.DISCORD_APPLICATION_ID;

    if (!token || !applicationId) {
        console.error('Missing DISCORD_BOT_TOKEN or DISCORD_APPLICATION_ID env variables.');
        process.exit(1);
    }

    const rest = new REST({ version: '10'}).setToken(token);
    try {
        console.log('Trying to register commands');
        const data = await rest.put(
            Routes.applicationCommands(applicationId),
            { body: commands }
        );

        console.log('Successfully registered ${commands.length} commands!');
        console.log('Commands: ', commands.map(x => x.name).join(', '));
    } catch (error) {
        console.error('Failed to register commands: ', error);
    }
}

registerCommands();