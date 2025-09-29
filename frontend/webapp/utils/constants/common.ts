export const IS_DEV = process.env.NODE_ENV === 'development';
const isLoopbackHost = typeof window !== 'undefined' ? /^(localhost|127\.0\.0\.1|\[::1\])$/.test(window.location.hostname) : false;

export const IS_LOCAL = IS_DEV && isLoopbackHost;
