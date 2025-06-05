import 'styled-components';

declare module 'styled-components' {
  export interface DefaultTheme {
    darkMode: boolean;
    colors: {
      // Custom Colors
      majestic_blue: string;
      majestic_blue_soft: string;
      orange_og: string;
      orange_soft: string;
      dark_red: string;
      darker_red: string;
      darkest_red: string;
      darkest_red_hover: string;
      dark_green: string;

      // Base Colors
      primary: string;
      secondary: string;
      border: string;
      dark_grey: string;
      translucent_bg: string;
      dropdown_bg: string;
      dropdown_bg_2: string;

      // Notification Colors
      warning: string;
      error: string;
      success: string;
      info: string;
      default: string;
    };
    text: {
      // Base Colors
      white: string;
      primary: string;
      secondary: string;
      grey: string;
      dark_grey: string;
      darker_grey: string;
      light_grey: string;
      dark_button: string;

      // Notification Colors
      warning: string;
      warning_secondary: string;
      error: string;
      error_secondary: string;
      success: string;
      success_secondary: string;
      info: string;
      info_secondary: string;
      default: string;
      default_secondary: string;
    };
    font_family: {
      primary: string;
      secondary: string;
      code: string;
    };
  }
}
