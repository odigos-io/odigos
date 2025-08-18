export const IS_DEV = process.env.NODE_ENV === 'development';
export const IS_LOCAL = IS_DEV && window.location.port === '3000';
