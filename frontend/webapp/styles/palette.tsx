import { DefaultTheme } from "styled-components";

// Define your color palette
const colors = {
  primary: "#FF0000",
  secondary: "#0EE6F3",
  torquiz_light: "#96F2FF",
  dark: "#07111A",
  light_dark: "#132330",
  dark_blue: "#203548",
  light_grey: "#CCD0D2",
  blue_grey: "#374A5B",
  white: "#fff",
  error: "#FD3F3F",
};

const text = {
  primary: "#07111A",
  secondary: "#0EE6F3",
  white: "#fff",
  light_grey: "#CCD0D2",
  grey: "#8b92a5",
  dark_button: "#0A1824",
};

const font_family = {
  primary: "Inter",
};

// Define the theme interface
interface ThemeInterface extends DefaultTheme {
  colors: typeof colors;
}

// Create your theme object
const theme: ThemeInterface = {
  colors,
  text,
  font_family,
};

// Export the theme
export default theme;
