// import { ApplicationCommandOptionType, ApplicationCommandType } from "discord-api-types/v10"

const PING_COMMAND = {
    name: 'ping',
    description: 'pong'
} as const;

export const commands = {
    ping: PING_COMMAND
} as const;
