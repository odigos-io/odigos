export const isEnterprise = (tier?: string) => ['onprem', 'enterprise'].includes(tier || '');
