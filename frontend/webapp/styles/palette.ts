import { DefaultTheme } from 'styled-components';

// Define your color palette
const colors = {
  primary: '#111',
  secondary: '#F9F9F9',
  dark_grey: '#151515',
  text: '#F9F9F9',

  torquiz_light: '#96F2FF',
  dark: '#07111A',
  light_dark: '#132330',
  dark_blue: '#203548',
  light_grey: '#CCD0D2',
  blue_grey: '#374A5B',
  white: '#fff',
  error: '#FD3F3F',
  orange_brown: '#b98a01',
  success: '#a5ff96',
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
  primary: 'Kode Mono, sans-serif',
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
