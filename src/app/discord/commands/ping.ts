import {
    SlashCommandBuilder,
    ChatInputCommandInteraction,
    InteractionResponse
} from 'discord.js';

export const data = new SlashCommandBuilder()
    .setName('ping')
    .setDescription('pong!');

export async function execute(interaction: ChatInputCommandInteraction): Promise<InteractionResponse> {
    return interaction.reply('pong');
}