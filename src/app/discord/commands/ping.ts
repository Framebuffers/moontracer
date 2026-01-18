export const data = {
  name: 'ping',
  description: 'Replies with Pong!',
};

export function execute(interaction: any) {
  return {
    type: 4,
    data: { content: 'pong!' }
  };
}