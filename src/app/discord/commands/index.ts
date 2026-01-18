import * as ping from './ping';
import * as awoo from './awoo';

export const commands = {
    ping,
    awoo
};

export const commandData = Object.values(commands).map(cmd => cmd.data);