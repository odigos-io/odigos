import { DefaultTheme } from 'styled-components';

// Define your color palette
const colors = {
  primary: '#111',
  secondary: '#F9F9F9',
  dark_grey: '#151515',
  text: '#F9F9F9',
  border: 'rgba(249, 249, 249, 0.08)',
  translucent_bg: '#1A1A1A',
  majestic_blue: '#444AD9',
};

const text = {
  primary: '#07111A',
  secondary: '#0EE6F3',
  white: '#fff',
  light_grey: '#CCD0D2',
  grey: '#8b92a5',
  dark_button: '#0A1824',
};

const font_family = {
  primary: 'Inter, sans-serif',
  secondary: 'Kode Mono, sans-serif',
};

// Define the theme interface
interface ThemeInterface extends DefaultTheme {
  colors: typeof colors;
  text: typeof text;
  font_family: typeof font_family;
}

// Create your theme object
const theme: ThemeInterface = {
  colors,
  text,
  font_family,
};

// Export the theme
export default theme;
