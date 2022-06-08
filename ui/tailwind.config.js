const defaultTheme = require("tailwindcss/defaultTheme");
module.exports = {
  content: [
    "./pages/**/*.{js,ts,jsx,tsx}",
    "./components/**/*.{js,ts,jsx,tsx}",
  ],
  fontFamily: {
    sans: ["Inter var", ...defaultTheme.fontFamily.sans],
    mono: ["Menlo", ...defaultTheme.fontFamily.mono],
    source: ["Source Sans Pro", ...defaultTheme.fontFamily.sans],
    "ubuntu-mono": ["Ubuntu Mono", ...defaultTheme.fontFamily.mono],
    system: defaultTheme.fontFamily.sans,
    flow: "Flow",
  },
  theme: {
    extend: {},
  },
  plugins: [require("@tailwindcss/forms")],
};
