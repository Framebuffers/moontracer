const DISCORD_BOT_TOKEN = process.env.DISCORD_BOT_TOKEN!;
const DISCORD_APPLICATION_ID = process.env.DISCORD_APPLICATION_ID;

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
    try {
        const url = `https://discord.com/api/v10/applications/${DISCORD_APPLICATION_ID}/commands`;

        const response = await fetch(url, {
            method: 'PUT',
            headers: {
                'Authorization': `Bot ${DISCORD_BOT_TOKEN}`,
                'Content-Type': 'application_json'
            },
            body: JSON.stringify(commands)
        });

        if (response.ok) {
            console.log('Command registered successfully!');
            const data = await response.json();
            console.log('Registered: ', data.map((x: any) => x.name).join(', '));
        } else {
            console.error('Failed to register commands');
            console.error(await response.text());
        }
    } catch (err) {
        console.error('Could not register commands: ', err)
    }
}

registerCommands();
