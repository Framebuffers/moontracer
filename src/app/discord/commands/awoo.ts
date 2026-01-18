export const data = {
  name: 'awoo',
  description: 'oh no, don\'t awoo.',
};

export function execute(interaction: any) {
  return {
    type: 4,
    data: { content: 'awooooooo' }
  };
}