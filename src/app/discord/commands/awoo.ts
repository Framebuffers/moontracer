import {
    SlashCommandBuilder,
    ChatInputCommandInteraction,
    InteractionResponse
} from 'discord.js';

export const data = new SlashCommandBuilder()
    .setName('awoo')
    .setDescription('oh no, don\'t awoo');

export async function execute(interaction: ChatInputCommandInteraction): Promise<InteractionResponse> {
    return interaction.reply('awoo');
}