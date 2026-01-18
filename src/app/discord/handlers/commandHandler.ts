import { commands } from '../commands';

export async function handleCommand(interaction: any) {
    const commandName = interaction.data.name as keyof typeof commands;
    const command = commands[commandName];

    if (!command || !command.execute) {
        return {
            type: 4,
            data: {
                content: 'Unknown command: ${commandName}',
                flags: 64 // ephemeral
            }
        };
    }

    return command.execute(interaction);
}