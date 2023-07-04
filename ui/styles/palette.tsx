import { DefaultTheme } from "styled-components";

// Define your color palette
const colors = {
  primary: "#FF0000",
  secondary: "#00FF00",
  accent: "#0000FF",
};

const text = {
  primary: "#FF0000",
  secondary: "#00FF00",
  white: "#fff",
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
